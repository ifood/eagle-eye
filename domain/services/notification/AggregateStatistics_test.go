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

package notification

import (
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"eagle-eye/mocks"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"testing"
	"time"
)

const (
	bucket string = "test"
	date   string = "2021-02-28"
)

func TestUpdateRemote(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockScanRepo := mocks.NewMockAggregateScanRepository(mockCtrl)

	scanResult := mockScanResult()

	mockScanRepo.EXPECT().GetByBucketAndDate(bucket, 28, 2).Return(mockScanResult(), nil).Times(1)
	mockScanRepo.EXPECT().Save(entities.MergeScanResults(scanResult, scanResult)).Times(1)

	a := NewAggregateStatistics(mockScanRepo, logging.NewDiscardLog())
	a.Update(scanResult)
	a.UpdateGlobal()
}

func mockScanResult() entities.ScanResult {
	parsedDate, _ := time.Parse("2006-01-02", date)
	scanID, _ := uuid.NewUUID()

	return entities.ScanResult{
		Bucket:   bucket,
		Scanned:  1,
		Bypassed: 1,
		Matches:  1,
		Errors:   1,
		Entropy: map[string]int{
			"0": 1,
			"1": 1,
			"2": 1,
			"3": 1,
			"4": 1,
			"5": 1,
			"6": 1,
			"7": 1,
			"8": 1,
		},
		ScanID:     scanID.String(),
		ResultType: entities.Aggregate,
		Requests:   1,
		LastUpdate: parsedDate,
	}
}
