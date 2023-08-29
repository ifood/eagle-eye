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
	"eagle-eye/logging"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProbabilisticFilter(t *testing.T) {
	probabilities := make(map[string]float64)
	probabilities["test"] = 0.0
	probabilities["test2"] = 1.0

	probabilisticFilter := NewProbabilisticFilter(probabilities, logging.NewDiscardLog())
	type test struct {
		request        entities.ScanRequest
		expectedResult entities.JobStatus
	}
	tests := []test{
		{request: entities.ScanRequest{Key: []string{"filename"}, Bucket: "test"}, expectedResult: entities.Abort},
		{request: entities.ScanRequest{Key: []string{"key2"}, Bucket: "test"}, expectedResult: entities.Abort},
		{request: entities.ScanRequest{Key: []string{"key3"}, Bucket: "test"}, expectedResult: entities.Abort},
		{request: entities.ScanRequest{Key: []string{"filename"}, Bucket: "test2"}, expectedResult: entities.NextJob},
		{request: entities.ScanRequest{Key: []string{"key2"}, Bucket: "test2"}, expectedResult: entities.NextJob},
		{request: entities.ScanRequest{Key: []string{"key3"}, Bucket: "test2"}, expectedResult: entities.NextJob},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expectedResult, probabilisticFilter.Filter(context.Background(), &tc.request))
	}
}
