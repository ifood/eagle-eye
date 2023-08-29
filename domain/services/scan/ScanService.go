/*
 *    Copyright 2023 iFood
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package scan

import (
	"context"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/fileutils"
	"eagle-eye/logging"
	"fmt"
	"github.com/spf13/afero"
	"io/fs"
	"reflect"
)

const (
	scanBufferSize = 1024 * 1024
	baseDir        = ""
)

type Service struct {
	localStorageFactory out.LocalStorageFactory
	logger              logging.Logger
	scanners            []SyncProcess
	asyncScanners       []AsyncProcess
}

func NewScanService(localStorageFactory out.LocalStorageFactory, scanners []SyncProcess, asyncScanners []AsyncProcess, logger logging.Logger) *Service {
	return &Service{localStorageFactory: localStorageFactory, logger: logger, scanners: scanners, asyncScanners: asyncScanners}
}

func (s *Service) Scan(ctx context.Context, request entities.ScanRequest) entities.ScanResult {
	storage, err := s.localStorageFactory.GetStorageFromID(request.StorageID)
	if err != nil {
		s.logger.Errorw("Failed to open local storage", "error", err, "StorageID", request.StorageID)
		return entities.ScanResult{}
	}

	scanContext := scanContext{ScanID: request.ScanID,
		Bucket:  request.Bucket,
		Key:     request.Key[0],
		Flags:   request.Flags,
		Storage: storage,
		Buffer:  make([]byte, scanBufferSize)}

	result := s.ScanRecursiveV2(ctx, scanContext)
	result.ScanID = request.ScanID
	result.ResultType = request.ResultType
	result.Requests++
	s.logger.Debugw("scan executed", "bucket", request.Bucket, "key", request.Key, "response", result)

	return result
}

func (s *Service) ScanRecursiveV2(ctx context.Context, scanContext scanContext) entities.ScanResult {
	result := entities.NewScanResult(scanContext.Bucket)

	err := afero.Walk(scanContext.Storage, baseDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if err := s.checkFileIsValid(scanContext.Storage, path); err != nil {
			s.logger.Errorw("could not validate file", "error", err, "filename", path, "context", scanContext)
			result.Errors++
			return nil
		}

		filetype, err := s.getFiletype(scanContext.Storage, path)
		if err != nil {
			s.logger.Errorw("failed to get filetype", "error", err, "filename", path, "context", scanContext)
			result.Errors++
			return nil
		}

		scanContext.Filetype = filetype
		scanContext.Filename = path

		partialResult := s.processFile(ctx, scanContext)
		result = entities.MergeScanResults(result, partialResult)

		return nil
	})

	if err != nil {
		s.logger.Errorw("error during scan", "error", err)
	}

	return result
}

//nolint:cyclop
func (s *Service) processFile(ctx context.Context, sc scanContext) (result entities.ScanResult) {
	switch sc.Filetype {
	case fileutils.Multimedia:
		// Images are being ignored because of high false positive rate.
		// In the future, we could analyze image histogram...
		return entities.ScanResult{Bypassed: 1}

	case fileutils.Executable, fileutils.Uncompressed:
		asyncResult := s.singleAsyncScan(ctx, sc)
		syncResult := s.singleSyncScan(ctx, sc)

		aggregateResult := entities.NewScanResult(sc.Bucket)
		aggregateResult = entities.MergeScanResults(aggregateResult, asyncResult)
		aggregateResult = entities.MergeScanResults(aggregateResult, syncResult)

		return aggregateResult

	case fileutils.Compressed:
		s.logger.Errorw("file was identified as compressed", "context", sc)
		return entities.ScanResult{Errors: 1}

	default:
		s.logger.Infow("unknown file being processed, should never happen", "context", sc)
		return entities.ScanResult{Errors: 1}
	}
}

func (s *Service) checkFileIsValid(storage out.LocalStorage, filename string) error {
	if exists, err := storage.Exists(filename); !exists || err != nil {
		return fmt.Errorf("file does not exist")
	}

	if regular, err := storage.IsRegular(filename); !regular || err != nil {
		return fmt.Errorf("file is a directory")
	}

	return nil
}

func (s *Service) getFiletype(storage out.LocalStorage, filename string) (fileutils.Filetype, error) {
	file, err := storage.Open(filename)
	if err != nil {
		return 0, fmt.Errorf("failed to open file. err: %w", err)
	}
	defer file.Close()

	filetype, err := fileutils.GetType(file)
	if err != nil {
		return 0, fmt.Errorf("failed to read type. err: %w", err)
	}

	return filetype, nil
}

func (s *Service) singleSyncScan(ctx context.Context, sc scanContext) entities.ScanResult {
	aggregateResult := entities.NewScanResult(sc.Bucket)
	aggregateResult.Scanned++

	for _, scanner := range s.scanners {
		s.logger.Debugw("Running scan job", "scanner type", reflect.ValueOf(scanner).Type())
		partialResult, err := scanner.Scan(ctx, sc)

		if err != nil {
			s.logger.Errorw("scan executed with error",
				"error", err,
				"scanner", reflect.ValueOf(scanner).Type(),
				"filename", sc.Filename, "bucket", sc.Bucket, "key", sc.Key)
		}

		aggregateResult = entities.MergeScanResults(aggregateResult, partialResult)
	}

	return aggregateResult
}

func (s *Service) singleAsyncScan(ctx context.Context, sc scanContext) entities.ScanResult {
	aggregateResult := entities.NewScanResult(sc.Bucket)

	for _, asyncScanner := range s.asyncScanners {
		s.logger.Debugw("Running scan job", "asyncScanner type", reflect.ValueOf(asyncScanner).Type())
		partialResult, err := asyncScanner.ScheduleScan(ctx, sc)

		if err != nil {
			s.logger.Errorw("async scan executed with error",
				"error", err,
				"asyncScanner", reflect.ValueOf(asyncScanner).Type(),
				"filename", sc.Filename, "bucket", sc.Bucket, "key", sc.Key)
		}

		aggregateResult = entities.MergeScanResults(aggregateResult, partialResult)
	}

	return aggregateResult
}
