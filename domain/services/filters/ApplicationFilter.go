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

package filters

import (
	"context"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/domain/services"
	"eagle-eye/fileutils"
	"eagle-eye/logging"
)

const HeaderSize uint64 = 1024

type ApplicationFilter struct {
	downloadService     services.Downloader
	localStorageFactory out.LocalStorageFactory
	logger              logging.Logger
}

func NewApplicationFilter(downloadService services.Downloader, localStorageFactory out.LocalStorageFactory, logger logging.Logger) *ApplicationFilter {
	return &ApplicationFilter{downloadService: downloadService, localStorageFactory: localStorageFactory, logger: logger}
}

func (a *ApplicationFilter) Filter(ctx context.Context, request *entities.ScanRequest) entities.JobStatus {
	isDownloadOk := a.downloadService.DownloadHeader(request, HeaderSize)
	if !isDownloadOk {
		return entities.Abort
	}

	localStorage, err := a.localStorageFactory.GetStorageFromID(request.StorageID)
	if err != nil {
		a.logger.Errorw("Failed to open local storage", "error", err, "bucket", request.Bucket, "key", request.Key[0])
		return entities.Abort
	}

	status := entities.Abort
	for index, filename := range request.Key {
		status = func() entities.JobStatus {
			file, err := localStorage.Open(filename)
			if err != nil {
				a.logger.Errorw("failed to open file locally", "error", err, "bucket", request.Bucket, "key", request.Key[index])
				return entities.Abort
			}
			defer file.Close()

			if fileutils.IsExecutable(file) {
				a.logger.Infow("Binary application detected on bucket, file will be scanned", "bucket", request.Bucket, "key", request.Key)
				return entities.NextStage
			}

			return entities.NextJob
		}()

		if status != entities.NextJob {
			return status
		}
	}

	/*
		Rationale: There are multiple backups which may contain executable files inside. Eg.: docker images.
		However, if we were to attempt to scan them, they would exhaust our allowed number of requests per day on VT.
		Of course, this is also a weakness.
	*/
	request.Flags |= entities.DisableVirusTotal

	return status
}
