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
	adapters "eagle-eye/adapters/out"
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"eagle-eye/mocks"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

const headerSize uint64 = 1024

func TestApplicationFilter(t *testing.T) {
	mockCtrl := gomock.NewController(t)

	mockDownloadService := mocks.NewMockDownloader(mockCtrl)
	mockLocalStorageFactory := mocks.NewMockLocalStorageFactory(mockCtrl)

	localStorageFactory := adapters.NewLocalStorageFactory(1024 * 1024 * 10)

	type test struct {
		filename       string
		expectedResult entities.JobStatus
	}

	tests := []test{
		{filename: "file_exe", expectedResult: entities.NextStage},
		{filename: "file_elf", expectedResult: entities.NextStage},
		{filename: "file_dll", expectedResult: entities.NextStage},
		{filename: "file_so", expectedResult: entities.NextStage},
		{filename: "file_macho", expectedResult: entities.NextStage},
		{filename: "file_unknown", expectedResult: entities.NextJob},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("Application filter for %s", tc.filename), func(t *testing.T) {
			memStorage, _ := localStorageFactory.GetLocalStorage(0, false)
			file, _ := memStorage.Create(tc.filename)
			common.LoadFileToStorage(t, tc.filename, file)

			scanRequest := &entities.ScanRequest{
				StorageType: "s3",
				Bucket:      "samples-scanner-sandbox",
				Key:         []string{tc.filename},
				Size:        1024,
				StorageID:   memStorage.GetID(),
			}

			mockDownloadService.EXPECT().DownloadHeader(scanRequest, headerSize).Return(true)
			mockLocalStorageFactory.EXPECT().GetStorageFromID(scanRequest.StorageID).Return(memStorage, nil)

			applicationFilter := NewApplicationFilter(mockDownloadService, mockLocalStorageFactory, logging.NewDiscardLog())
			assert.Equal(t, tc.expectedResult, applicationFilter.Filter(context.Background(), scanRequest))
		})
	}
}
