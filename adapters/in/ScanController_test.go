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

package in

import (
	"bytes"
	adapterentities "eagle-eye/adapters/entities"
	"eagle-eye/common"
	http2 "eagle-eye/http"
	"eagle-eye/logging"
	"eagle-eye/mocks"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidFileForScan(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockScheduler := mocks.NewMockScheduler(mockCtrl)
	mockScheduler.EXPECT().Schedule("fakename", gomock.Any()).Return("scanid", nil).Times(1)
	scanController := NewScanController(mockScheduler, logging.NewDiscardLog())

	handlers := []http2.Handler{
		{HTTPMethod: "POST", Path: "/files", HandlerFunc: scanController.ScanFile},
	}
	app := common.CreateFiberAppForTest(handlers)
	body, contentType := common.PrepareRequestBody(t, "file", []byte{0xca, 0xfe, 0xba, 0xbe})

	request := httptest.NewRequest("POST", "/v1/files", body)
	request.Header.Add("Content-type", contentType)

	httpResponse, err := app.Test(request, -1)
	if err != nil {
		t.Errorf("failed to send request. %v", err)
	}
	defer httpResponse.Body.Close()

	decoder := json.NewDecoder(httpResponse.Body)

	var scheduleResponse adapterentities.ScheduleResponse
	err = decoder.Decode(&scheduleResponse)
	assert.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, httpResponse.StatusCode)
	assert.NotEmpty(t, scheduleResponse.ID)
	assert.Empty(t, scheduleResponse.Error)
}

func TestValidObjectForScan(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockScheduler := mocks.NewMockScheduler(mockCtrl)
	mockScheduler.EXPECT().ScheduleObject("sample-bucket", "sample-file").Return("scanid", nil).Times(1)
	scanController := NewScanController(mockScheduler, logging.NewDiscardLog())

	handlers := []http2.Handler{
		{HTTPMethod: "POST", Path: "/files", HandlerFunc: scanController.ScanFile},
		{HTTPMethod: "POST", Path: "/objects", HandlerFunc: scanController.ScanObject},
	}
	app := common.CreateFiberAppForTest(handlers)

	body := "{\"region\":\"us-east-1\",\"bucket\":\"sample-bucket\",\"key\":\"sample-file\"}"
	request := httptest.NewRequest("POST", "/v1/objects", strings.NewReader(body))
	request.Header.Add("Content-type", "application/json")

	httpResponse, err := app.Test(request, -1)
	if err != nil {
		t.Errorf("failed to send request. %v", err)
	}
	defer httpResponse.Body.Close()

	var scheduleResponse adapterentities.ScheduleResponse
	decoder := json.NewDecoder(httpResponse.Body)
	err = decoder.Decode(&scheduleResponse)
	assert.NoError(t, err)

	assert.Equal(t, fiber.StatusOK, httpResponse.StatusCode)
	assert.NotEmpty(t, scheduleResponse.ID)
	assert.Empty(t, scheduleResponse.Error)
}

func TestInvalidObjectRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockScheduler := mocks.NewMockScheduler(mockCtrl)
	scanController := NewScanController(mockScheduler, logging.NewDiscardLog())

	handlers := []http2.Handler{
		{HTTPMethod: "POST", Path: "/files", HandlerFunc: scanController.ScanFile},
		{HTTPMethod: "POST", Path: "/objects", HandlerFunc: scanController.ScanObject},
	}
	app := common.CreateFiberAppForTest(handlers)

	tests := []struct {
		TestName string
		Body     string
	}{
		{TestName: "missing required fields", Body: "{\"xxxx\":\"yyyy\"}"},
		{TestName: "invalid body type", Body: "invalid json"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.TestName, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/v1/objects", strings.NewReader(test.Body))
			request.Header.Add("Content-type", "application/json")

			httpResponse, err := app.Test(request, -1)
			if err != nil {
				t.Errorf("failed to send request. %v", err)
			}
			defer httpResponse.Body.Close()

			var scheduleResponse adapterentities.ScheduleResponse
			decoder := json.NewDecoder(httpResponse.Body)
			err = decoder.Decode(&scheduleResponse)
			assert.NoError(t, err)

			assert.Equal(t, fiber.StatusBadRequest, httpResponse.StatusCode)
			assert.Empty(t, scheduleResponse.ID)
			assert.NotEmpty(t, scheduleResponse.Error)
		})
	}
}

func TestInvalidRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockScheduler := mocks.NewMockScheduler(mockCtrl)
	scanController := NewScanController(mockScheduler, logging.NewDiscardLog())

	handlers := []http2.Handler{
		{HTTPMethod: "POST", Path: "/files", HandlerFunc: scanController.ScanFile},
		{HTTPMethod: "POST", Path: "/objects", HandlerFunc: scanController.ScanObject},
	}
	app := common.CreateFiberAppForTest(handlers)
	var scheduleResponse adapterentities.ScheduleResponse

	t.Run("invalid content type", func(t *testing.T) {
		body, _ := common.PrepareRequestBody(t, "file", []byte{0xca, 0xfe, 0xba, 0xbe})
		request := httptest.NewRequest("POST", "/v1/files", body)
		request.Header.Add("Content-type", "invalidtype")

		httpResponse, err := app.Test(request, -1)
		if err != nil {
			t.Errorf("failed to send request. %v", err)
		}
		defer httpResponse.Body.Close()

		decoder := json.NewDecoder(httpResponse.Body)
		err = decoder.Decode(&scheduleResponse)
		assert.NoError(t, err)

		assert.Equal(t, fiber.StatusBadRequest, httpResponse.StatusCode)
		assert.Empty(t, scheduleResponse.ID)
		assert.NotEmpty(t, scheduleResponse.Error)
	})

	t.Run("incorrect request", func(t *testing.T) {
		request := httptest.NewRequest("POST", "/v1/files", bytes.NewReader([]byte{}))

		httpResponse, err := app.Test(request, -1)
		if err != nil {
			t.Errorf("failed to send request. %v", err)
		}
		defer httpResponse.Body.Close()

		decoder := json.NewDecoder(httpResponse.Body)
		err = decoder.Decode(&scheduleResponse)
		assert.NoError(t, err)

		assert.Equal(t, fiber.StatusBadRequest, httpResponse.StatusCode)
		assert.Empty(t, scheduleResponse.ID)
		assert.NotEmpty(t, scheduleResponse.Error)
	})
}
