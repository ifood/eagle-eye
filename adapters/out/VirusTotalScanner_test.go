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
	"context"
	"eagle-eye/adapters/entities"
	out2 "eagle-eye/domain/ports/out"
	"eagle-eye/mocks"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestHttpErrorOnCall(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRateLimiter := mocks.NewMockRateLimiter(mockCtrl)
	mockRateLimiter.EXPECT().IsRequestAllowed().Return(true)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", fmt.Sprintf("=~%s", scanHashURL),
		httpmock.NewStringResponder(http.StatusInternalServerError, ""))

	r := NewVirusTotalScanner("DUMMY_KEY", 10.0, mockRateLimiter)
	result := r.ScanHash(context.Background(), "a7bbc4b4f781e04214ecebe69a766c76681aa7eb")

	assert.Equal(t, out2.Error, result.AnalysisResult)
}

func TestNoDetectionBelowThreshold(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRateLimiter := mocks.NewMockRateLimiter(mockCtrl)
	mockRateLimiter.EXPECT().IsRequestAllowed().Return(true)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	vtResult := mockVirusTotalResponse(1, 10)

	httpmock.RegisterResponder("GET", fmt.Sprintf("=~%s", scanHashURL),
		httpmock.NewJsonResponderOrPanic(http.StatusOK, vtResult))

	r := NewVirusTotalScanner("DUMMY_KEY", 10.0, mockRateLimiter)
	result := r.ScanHash(context.Background(), "a7bbc4b4f781e04214ecebe69a766c76681aa7eb")

	assert.Equal(t, out2.Benign, result.AnalysisResult)
}

func TestDetectionAboveThreshold(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	mockRateLimiter := mocks.NewMockRateLimiter(mockCtrl)
	mockRateLimiter.EXPECT().IsRequestAllowed().Return(true)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	vtResult := mockVirusTotalResponse(10, 0)

	httpmock.RegisterResponder("GET", fmt.Sprintf("=~%s", scanHashURL),
		httpmock.NewJsonResponderOrPanic(http.StatusOK, vtResult))

	r := NewVirusTotalScanner("DUMMY_KEY", 10.0, mockRateLimiter)
	result := r.ScanHash(context.Background(), "a7bbc4b4f781e04214ecebe69a766c76681aa7eb")

	assert.Equal(t, out2.Malicious, result.AnalysisResult)
}

func mockVirusTotalResponse(malicious, undetected int) entities.VTScanResult {
	return entities.VTScanResult{
		Data: entities.Data{
			Attributes: entities.Attributes{
				LastAnalysisStats: entities.AnalysisStats{
					Malicious:  malicious,
					Undetected: undetected,
				},
			},
		},
	}
}
