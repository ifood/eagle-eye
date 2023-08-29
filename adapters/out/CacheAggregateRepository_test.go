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

package out

import (
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"eagle-eye/mocks"
	"encoding/json"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSave(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	result := entities.NewScanResult("test")
	jsonResult, _ := json.Marshal(result)

	mockCache := mocks.NewMockCache(mockCtrl)
	mockCache.EXPECT().Lock(gomock.Any(), gomock.Any()).AnyTimes()
	mockCache.EXPECT().Unlock(gomock.Any()).AnyTimes()
	mockCache.EXPECT().Set(gomock.Any(), string(jsonResult), gomock.Any()).Times(1)
	repo := NewCacheAggregateRepo(mockCache, logging.NewDiscardLog())

	err := repo.Save(result)
	assert.NoError(t, err)
}

func TestGetByMonth(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCache := mocks.NewMockCache(mockCtrl)
	scanResults := common.GetObjectsFromJSONFile[[]entities.ScanResult](t, "bucket_scan_data.json")
	expected := common.GetObjectsFromJSONFile[map[string]entities.ScanResult](t, "bucket_expected_result.json")

	repo := NewCacheAggregateRepo(mockCache, logging.NewDiscardLog())
	keys := make([]string, len(scanResults))

	for index, result := range scanResults {
		keys[index] = repo.getItemKey(result.LastUpdate.Day(), int(result.LastUpdate.Month()), result.Bucket)
		mockCache.EXPECT().Get(keys[index]).Return(common.GetObjectJSON(t, result), nil).Times(1)
	}

	mockCache.EXPECT().List(gomock.Any()).Return(keys, nil)

	result, err := repo.GetByMonth(1)

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestGetResultByBucket(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	expected := common.GetObjectsFromJSONFile[[]entities.ScanResult](t, "bucket_scan_data.json")[2]

	mockCache := mocks.NewMockCache(mockCtrl)
	mockCache.EXPECT().Lock(gomock.Any(), gomock.Any()).AnyTimes()
	mockCache.EXPECT().Unlock(gomock.Any()).AnyTimes()
	mockCache.EXPECT().Get(gomock.Any()).Return(common.GetObjectJSON(t, expected), nil).Times(1)

	repo := NewCacheAggregateRepo(mockCache, logging.NewDiscardLog())
	result, err := repo.GetByBucketAndDate(expected.Bucket, expected.LastUpdate.Day(), int(expected.LastUpdate.Month()))

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}
