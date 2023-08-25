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

package services

import (
	"eagle-eye/domain/entities"
	ports "eagle-eye/domain/ports/out"
	"eagle-eye/logging"
)

//go:generate go run -mod=mod github.com/golang/mock/mockgen -destination=../../mocks/mock_download_service.go -package=mocks -source=DownloadService.go

type Downloader interface {
	DownloadHeader(request *entities.ScanRequest, headerSize uint64) bool
	DownloadSingleFile(request *entities.ScanRequest) bool
}

type DownloadService struct {
	localStorageFactory  ports.LocalStorageFactory
	remoteStorageFactory ports.RemoteStorageFactory
	logger               logging.Logger
}

func NewDownloadService(localStorageFactory ports.LocalStorageFactory, remoteStorageFactory ports.RemoteStorageFactory, logger logging.Logger) Downloader {
	return &DownloadService{localStorageFactory: localStorageFactory, remoteStorageFactory: remoteStorageFactory, logger: logger}
}

func (d *DownloadService) DownloadHeader(request *entities.ScanRequest, headerSize uint64) bool {
	localStorage, err := d.localStorageFactory.GetStorageFromID(request.StorageID)
	if err != nil {
		d.logger.Errorw("Failed to get local storage", "error", err, "request", request)
		return false
	}

	isDownloadOk := true

	for _, key := range request.Key {
		isDownloadOk = isDownloadOk && func() bool {
			localFile, err := localStorage.Create(key)
			if err != nil {
				d.logger.Errorw("Failed to create local file", "error", err)
				return false
			}
			defer localFile.Close()

			// Downloads the remote file
			remoteStorage, err := d.remoteStorageFactory.GetRemoteStorage(request.StorageType)
			if err != nil {
				d.logger.Errorw("Failed to get remote storage", "error", err)
				return false
			}

			err = remoteStorage.GetHeader(request.Bucket, key, headerSize, localFile)
			if err != nil {
				d.logger.Errorw("Failed to request key from bucket", "error", err, "bucket", request.Bucket, "key", key)
				return false
			}

			return true
		}()
	}

	return isDownloadOk
}

func (d *DownloadService) DownloadSingleFile(request *entities.ScanRequest) bool {
	localStorage, err := d.localStorageFactory.GetStorageFromID(request.StorageID)
	if err != nil {
		d.logger.Errorw("Failed to get local storage", "error", err, "bucket", request.Bucket, "key", request.Key[0])
		return false
	}

	isDownloadOk := true

	for _, key := range request.Key {
		isDownloadOk = isDownloadOk && func() bool {
			localFile, _ := localStorage.Create(key)
			defer localFile.Close()

			// Downloads the remote file
			remoteStorage, err := d.remoteStorageFactory.GetRemoteStorage(request.StorageType)
			if err != nil {
				d.logger.Errorw("Failed to get remote storage", "error", err)
				return false
			}

			err = remoteStorage.Get(request.Bucket, key, localFile)
			if err != nil {
				d.logger.Errorw("Failed to request key from bucket", "error", err, "bucket", request.Bucket, "key", key)
				return false
			}

			return true
		}()
	}

	return isDownloadOk
}
