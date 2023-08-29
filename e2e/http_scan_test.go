//go:build e2e

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

package e2e

import (
	"context"
	adapterentities "eagle-eye/adapters/entities"
	"eagle-eye/app"
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"io"
	"net/http"
	"reflect"
	"time"
)

// Disable test because of concurrency issue
func (suite *E2E) TestHTTPScan() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		app.Start(ctx)
	}()

	suite.Require().Eventually(func() bool {
		resp, err := http.Get("http://localhost:3000/healthcheck/readiness")
		if err != nil {
			return false
		}
		return resp.StatusCode == fiber.StatusOK
	}, time.Minute, 5*time.Second)

	file := common.LoadFile(suite.T(), "cabeca_batata.jpeg")
	body, contentType := common.PrepareRequestBody(suite.T(), "file", file)

	request, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:3000/v1/files", body)
	request.Header.Add("Content-type", contentType)
	client := &http.Client{}

	httpResponse, err := client.Do(request)
	suite.Require().NoError(err)
	suite.Require().Equal(http.StatusOK, httpResponse.StatusCode)
	defer httpResponse.Body.Close()

	var scheduleResponse adapterentities.ScheduleResponse
	err = json.NewDecoder(httpResponse.Body).Decode(&scheduleResponse)
	suite.Assert().NoError(err)

	scanId := scheduleResponse.ID

	expected := map[string]adapterentities.ScanResponse{
		scanId: {
			Bucket:   "no bucket specified",
			Scanned:  0,
			Bypassed: 1,
			Matches:  0,
			Errors:   0,
			Entropy:  entities.GenerateEntropyBuckets([9]int{0, 0, 0, 0, 0, 0, 0, 0, 0}),
			Requests: 1,
		},
	}

	request, _ = http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://localhost:3000/v1/files/%s", scanId), http.NoBody)
	suite.Require().Eventually(func() bool {
		httpResponse, err := client.Do(request)
		if err != nil || httpResponse.StatusCode != http.StatusOK {
			return false
		}

		var obtained adapterentities.ObjectScanResponse
		body, err := io.ReadAll(httpResponse.Body)
		if err != nil {
			return false
		}

		err = json.Unmarshal(body, &obtained)
		if err != nil {
			return false
		}

		fmt.Printf("%#v", obtained)

		return reflect.DeepEqual(expected, obtained.Result)
	}, 2*time.Minute, 10*time.Second)
}
