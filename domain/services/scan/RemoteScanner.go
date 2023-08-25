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
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/fileutils"
	"eagle-eye/logging"
	"fmt"
	"io"
	"time"
)

const (
	timeLimit = -1 * time.Hour
)

type RemoteScanner struct {
	scheduleScanRepository out.ScheduleScanRepository[entities.ScheduleItem]
	scanner                out.RemoteScan
	queryInterval          time.Duration
	logger                 logging.Logger
}

func NewRemoteScanner(scanner out.RemoteScan, queryInterval time.Duration, scheduleScanRepository out.ScheduleScanRepository[entities.ScheduleItem], logger logging.Logger) *RemoteScanner {
	if !scanner.IsAvailable() {
		logger.Infow("Remote scanner was not properly configured. Therefore, it won't be used for file scan of executable files.")
	}
	return &RemoteScanner{scheduleScanRepository: scheduleScanRepository, scanner: scanner, queryInterval: queryInterval, logger: logger}
}

func (r *RemoteScanner) ScheduleScan(ctx context.Context, sc scanContext) (entities.ScanResult, error) {
	if !r.scanner.IsAvailable() || !r.shouldScan(sc) {
		return entities.ScanResult{}, nil
	}

	file, err := sc.Storage.Open(sc.Filename)
	if err != nil {
		return entities.ScanResult{Errors: 1}, fmt.Errorf("failed to open file")
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return entities.ScanResult{Errors: 1}, fmt.Errorf("failed to read data from local file. err: %w", err)
	}

	// Scanning the binary
	remoteResult := r.scanner.ScanBinary(ctx, data)
	if remoteResult.AnalysisResult != out.InProgress {
		return entities.ScanResult{Errors: 1}, fmt.Errorf("failed to query external service. err: %w", err)
	}

	err = r.scheduleScanRepository.Add(remoteResult.ID, entities.ScheduleItem{
		ScanID:     remoteResult.ID,
		CreateTime: time.Now(),
		Bucket:     sc.Bucket,
		Key:        sc.Key,
		Filename:   sc.Filename,
	})
	if err != nil {
		return entities.ScanResult{Errors: 1}, err
	}

	return entities.ScanResult{}, nil
}

func (r *RemoteScanner) shouldScan(scanContext scanContext) bool {
	if scanContext.Filetype != fileutils.Executable {
		return false
	}

	if scanContext.Flags&entities.DisableVirusTotal != 0 {
		return false
	}

	return true
}

func (r *RemoteScanner) GetResults(ctx context.Context) []entities.ScanResult {
	var results []entities.ScanResult

	scheduleItems, errors := r.scheduleScanRepository.GetUntil(time.Now().Add(-r.queryInterval))
	for _, err := range errors {
		r.logger.Errorw("failed to obtain some of the schedule information", "error", err)
	}

	r.logger.Debugw("Obtained scan ids for querying", "Total", len(scheduleItems))

	expirationTimeLimit := time.Now().Add(timeLimit)
	for _, scheduleItem := range scheduleItems {
		if scheduleItem.CreateTime.Before(expirationTimeLimit) {
			r.logger.Errorw("Handle id was queried and hasn't executed yet. Because its over our time limit, it'll be ignored.", "item", scheduleItem)
			continue
		}

		status := r.scanner.GetScanResult(ctx, scheduleItem.ScanID)
		switch status.AnalysisResult {
		case out.InProgress:
			r.logger.Infow("Handle id not yet ready. Reinserting to schedule repo", "scanID", scheduleItem.ScanID)

			if err := r.scheduleScanRepository.Add(scheduleItem.ScanID, scheduleItem); err != nil {
				r.logger.Errorw("Failed to save schedule item", "error", err, "item", scheduleItem)
			}
		case out.Benign:
			r.logger.Debugw("File was marked as undetected, check scanid.", "item", scheduleItem)
		case out.Malicious:
			r.logger.Infow("File was detected as malicious, check scanid.", "item", scheduleItem)

			results = append(results, entities.ScanResult{Bucket: scheduleItem.Bucket, Matches: 1, Entropy: common.CreateEmptyEntropyBuckets(), LastUpdate: time.Now()})
		case out.Error, out.Unseen, out.DecodeError, out.InvalidID:
			r.logger.Errorw("Failed to get output from remote repo", "item", scheduleItem)
		}
	}

	return results
}
