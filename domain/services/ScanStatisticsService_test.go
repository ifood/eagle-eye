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
	"eagle-eye/domain/ports/out"
	"eagle-eye/logging"
	"eagle-eye/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGetSingleBucket(t *testing.T) {
	t.Run("get daily result from single bucket", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockAggRepo := mocks.NewMockAggregateScanRepository(mockCtrl)
		mockIndividualRepo := mocks.NewMockIndividualScanRepository(mockCtrl)
		mockScheduler := mocks.NewMockScheduler(mockCtrl)
		externalStatistics := NewScanStatisticsService(mockAggRepo, mockIndividualRepo, mockScheduler, map[entities.ViewerMimetype]out.Viewer{}, logging.NewDiscardLog())
		today := time.Now()
		response := getMockResult(today)
		mockAggRepo.EXPECT().GetByDate(today.Day(), int(today.Month())).Return(response, nil).Times(1)

		obtainedResult, err := externalStatistics.GetBucketsStatistics("sample-bucket", today, "day")

		assert.NoError(t, err)
		assert.Equal(t, map[string]entities.ScanResult{"sample-bucket": response["sample-bucket"]}, obtainedResult)
	})

	t.Run("get monthly result from single bucket", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockAggRepo := mocks.NewMockAggregateScanRepository(mockCtrl)
		mockIndividualRepo := mocks.NewMockIndividualScanRepository(mockCtrl)
		mockScheduler := mocks.NewMockScheduler(mockCtrl)
		externalStatistics := NewScanStatisticsService(mockAggRepo, mockIndividualRepo, mockScheduler, map[entities.ViewerMimetype]out.Viewer{}, logging.NewDiscardLog())
		today := time.Now()
		expectedResponse := getMockResult(today)

		mockAggRepo.EXPECT().GetByMonth(int(today.Month())).Return(expectedResponse, nil).Times(1)
		obtainedResult, err := externalStatistics.GetBucketsStatistics("sample-bucket", today, "month")
		assert.NoError(t, err)
		assert.Equal(t, map[string]entities.ScanResult{"sample-bucket": expectedResponse["sample-bucket"]}, obtainedResult)
	})

	t.Run("get result from unknown bucket", func(t *testing.T) {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockAggRepo := mocks.NewMockAggregateScanRepository(mockCtrl)
		mockIndividualRepo := mocks.NewMockIndividualScanRepository(mockCtrl)
		mockScheduler := mocks.NewMockScheduler(mockCtrl)
		externalStatistics := NewScanStatisticsService(mockAggRepo, mockIndividualRepo, mockScheduler, map[entities.ViewerMimetype]out.Viewer{}, logging.NewDiscardLog())
		today := time.Now()
		response := getMockResult(today)

		mockAggRepo.EXPECT().GetByDate(today.Day(), int(today.Month())).Return(response, nil).Times(1)

		_, err := externalStatistics.GetBucketsStatistics("unknown", today, "day")
		assert.Error(t, err)
	})
}

func TestViewer(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	const TestType entities.ViewerMimetype = "TestType"
	mockViewer := mocks.NewMockViewer(mockCtrl)
	mockAggRepo := mocks.NewMockAggregateScanRepository(mockCtrl)
	mockIndividualRepo := mocks.NewMockIndividualScanRepository(mockCtrl)
	mockScheduler := mocks.NewMockScheduler(mockCtrl)
	externalStatistics := NewScanStatisticsService(mockAggRepo, mockIndividualRepo, mockScheduler, map[entities.ViewerMimetype]out.Viewer{TestType: mockViewer}, logging.NewDiscardLog())
	today := time.Now()
	response := getMockResult(today)

	mockAggRepo.EXPECT().GetByMonth(int(today.Month())).Return(response, nil).Times(1)
	mockViewer.EXPECT().Show(gomock.Any(), response).Times(1)
	externalStatistics.Show(TestType, NoBucketName, today, "month")
}

func getMockResult(today time.Time) map[string]entities.ScanResult {
	return map[string]entities.ScanResult{
		"sample-bucket":    {ScanID: "", Bucket: "sample-bucket", Scanned: 10, Bypassed: 1, Errors: 2, Matches: 3, Requests: 4, LastUpdate: today},
		"sample-bucket-v2": {ScanID: "", Bucket: "sample-bucket-v2", Scanned: 13, Bypassed: 5, Errors: 1, Matches: 4, Requests: 9, LastUpdate: today},
	}
}
