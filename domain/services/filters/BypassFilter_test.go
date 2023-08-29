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

func TestBypass(t *testing.T) {
	allowList := make(map[string][]string)
	allowList["test"] = []string{"folder1/", "folder2/"}
	sizeLimit := uint64(1024 * 1024)

	bypassFilter := NewBypassfilter(allowList, sizeLimit, logging.NewDiscardLog())

	type test struct {
		request        entities.ScanRequest
		expectedResult entities.JobStatus
	}

	tests := []test{
		{request: entities.ScanRequest{Key: []string{"test"}, Bucket: "test"}, expectedResult: entities.NextJob},
		{request: entities.ScanRequest{Key: []string{"folder2diff/test1"}, Bucket: "test"}, expectedResult: entities.NextJob},
		{request: entities.ScanRequest{Key: []string{"folder1/test2/test1"}, Bucket: "test"}, expectedResult: entities.Abort},
		{request: entities.ScanRequest{Key: []string{"folder1/test1"}, Bucket: "test"}, expectedResult: entities.Abort},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expectedResult, bypassFilter.Filter(context.Background(), &tc.request))
	}
}
