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

package preprocess

import (
	"context"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/services"
	"reflect"
)

type Downloader struct {
	downloadService services.Downloader
}

func NewDownloader(downloadService services.Downloader) *Downloader {
	return &Downloader{downloadService: downloadService}
}

func (d *Downloader) Preprocess(ctx context.Context, request *entities.ScanRequest) entities.JobStatus {
	if d.downloadService.DownloadSingleFile(request) {
		return entities.NextJob
	}

	return entities.Abort
}

func (d *Downloader) Name() string {
	return reflect.TypeOf(d).Name()
}
