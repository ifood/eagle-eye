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
	adapterentities "eagle-eye/adapters/entities"
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/services"
	http2 "eagle-eye/http"
	"eagle-eye/logging"
	"eagle-eye/mocks"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAggregateGetResult(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	result := entities.NewScanResult("teste")
	buckets := map[string]entities.ScanResult{"teste": result}
	expected := adapterentities.ObjectScanResponse{Result: adapterentities.MapResultToScanResponse(buckets)}
	mockStatisticsService := mocks.NewMockStatisticsService(mockCtrl)
	mockStatisticsService.EXPECT().GetBucketsStatistics(gomock.Any(), gomock.Any(), gomock.Any()).Return(buckets, nil).AnyTimes()

	statisticsController := NewStatisticsController(mockStatisticsService, logging.NewDiscardLog())
	handlers := []http2.Handler{
		{HTTPMethod: "GET", Path: "/objects", HandlerFunc: statisticsController.GetAggregateResult},
	}
	app := common.CreateFiberAppForTest(handlers)

	request := httptest.NewRequest("GET", "/v1/objects", http.NoBody)

	response, err := app.Test(request, -1)
	if err != nil {
		t.Errorf("failed to send request. %v", err)
	}
	defer response.Body.Close()

	assert.Equal(t, fiber.StatusOK, response.StatusCode)

	body, err := io.ReadAll(response.Body)
	assert.NoError(t, err)

	obtained := common.GetObjectFromJSON[adapterentities.ObjectScanResponse](t, body)
	assert.Equal(t, expected, obtained)
}

func TestInvalidAggregateRequest(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockStatisticsService := mocks.NewMockStatisticsService(mockCtrl)
	statisticsController := NewStatisticsController(mockStatisticsService, logging.NewDiscardLog())
	handlers := []http2.Handler{
		{HTTPMethod: "GET", Path: "/objects", HandlerFunc: statisticsController.GetAggregateResult},
	}
	app := common.CreateFiberAppForTest(handlers)

	t.Run("invalid period", func(t *testing.T) {
		request := httptest.NewRequest("GET", "/v1/objects?period=invalidperiod", http.NoBody)

		response, err := app.Test(request, -1)
		if err != nil {
			t.Errorf("failed to send request. %v", err)
		}
		defer response.Body.Close()

		assert.Equal(t, fiber.StatusBadRequest, response.StatusCode)

		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)

		obtained := common.GetObjectFromJSON[adapterentities.ObjectScanResponse](t, body)
		assert.NotEmpty(t, obtained.Error)
		assert.Empty(t, obtained.Result)
	})

	t.Run("invalid date", func(t *testing.T) {
		request := httptest.NewRequest("GET", "/v1/objects?date=2022-13-01", http.NoBody)

		response, err := app.Test(request, -1)
		if err != nil {
			t.Errorf("failed to send request. %v", err)
		}
		defer response.Body.Close()

		assert.Equal(t, fiber.StatusBadRequest, response.StatusCode)

		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)

		obtained := common.GetObjectFromJSON[adapterentities.ObjectScanResponse](t, body)
		assert.NotEmpty(t, obtained.Error)
		assert.Empty(t, obtained.Result)
	})

	t.Run("invalid date format", func(t *testing.T) {
		request := httptest.NewRequest("GET", "/v1/objects?date=20-01-01", http.NoBody)

		response, err := app.Test(request, -1)
		if err != nil {
			t.Errorf("failed to send request. %v", err)
		}
		defer response.Body.Close()

		assert.Equal(t, fiber.StatusBadRequest, response.StatusCode)

		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)

		obtained := common.GetObjectFromJSON[adapterentities.ObjectScanResponse](t, body)
		assert.NotEmpty(t, obtained.Error)
		assert.Empty(t, obtained.Result)
	})
}

func TestIndividualGetResult(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	scanID, err := uuid.NewRandom()
	assert.NoError(t, err)

	expectedOutput := entities.ScanResult{ScanID: scanID.String()}
	mockStatisticsService := mocks.NewMockStatisticsService(mockCtrl)
	mockStatisticsService.EXPECT().GetScanResult(scanID.String()).Return(expectedOutput, nil)
	statisticsController := NewStatisticsController(mockStatisticsService, logging.NewDiscardLog())
	handlers := []http2.Handler{
		{HTTPMethod: "GET", Path: "/files/:id", HandlerFunc: statisticsController.GetFileResult},
	}
	app := common.CreateFiberAppForTest(handlers)

	request := httptest.NewRequest("GET", fmt.Sprintf("/v1/files/%s", scanID), http.NoBody)

	response, err := app.Test(request, -1)
	if err != nil {
		t.Errorf("failed to send request. %v", err)
	}
	defer response.Body.Close()

	assert.Equal(t, fiber.StatusOK, response.StatusCode)
}

func TestIndividualScanErrors(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	t.Run("invalid uuid", func(t *testing.T) {
		mockStatisticsService := mocks.NewMockStatisticsService(mockCtrl)
		statisticsController := NewStatisticsController(mockStatisticsService, logging.NewDiscardLog())
		handlers := []http2.Handler{
			{HTTPMethod: "GET", Path: "/files/:id", HandlerFunc: statisticsController.GetFileResult},
		}
		app := common.CreateFiberAppForTest(handlers)

		request := httptest.NewRequest("GET", fmt.Sprintf("/v1/files/%s", "xxxx"), http.NoBody)

		response, err := app.Test(request, -1)
		if err != nil {
			t.Errorf("failed to send request. %v", err)
		}
		defer response.Body.Close()

		assert.Equal(t, fiber.StatusBadRequest, response.StatusCode)

		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)

		obtained := common.GetObjectFromJSON[adapterentities.ObjectScanResponse](t, body)
		assert.NotEmpty(t, obtained.Error)
		assert.Empty(t, obtained.Result)
	})

	t.Run("internal error has occurred", func(t *testing.T) {
		mockStatisticsService := mocks.NewMockStatisticsService(mockCtrl)
		mockStatisticsService.EXPECT().GetScanResult(gomock.Any()).Return(entities.ScanResult{}, services.ErrScanIDNotFound)
		statisticsController := NewStatisticsController(mockStatisticsService, logging.NewDiscardLog())
		handlers := []http2.Handler{
			{HTTPMethod: "GET", Path: "/files/:id", HandlerFunc: statisticsController.GetFileResult},
		}
		app := common.CreateFiberAppForTest(handlers)

		request := httptest.NewRequest("GET", fmt.Sprintf("/v1/files/%s", uuid.NewString()), http.NoBody)

		response, err := app.Test(request, -1)
		if err != nil {
			t.Errorf("failed to send request. %v", err)
		}
		defer response.Body.Close()

		assert.Equal(t, fiber.StatusNotFound, response.StatusCode)

		body, err := io.ReadAll(response.Body)
		assert.NoError(t, err)

		obtained := common.GetObjectFromJSON[adapterentities.ObjectScanResponse](t, body)
		assert.NotEmpty(t, obtained.Error)
		assert.Empty(t, obtained.Result)
	})
}
