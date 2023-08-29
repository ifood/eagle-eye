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
	"eagle-eye/domain/entities"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"io"
	"net/http"
	"reflect"
	"time"
)

func (suite *E2E) TestQueueScan() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	files := []string{"[DANGEROUS]_emotet", "cabeca_batata.jpeg", "emptyfile", "file_dll", "file_elf", "file_exe", "file_macho", "file_so",
		"file_unknown", "monke.png", "pepe_hack.gif", "repo.bundle", "text.gz", "text.lz4", "text.txt", "text.txt.gz", "text.zip"}

	for _, filepath := range files {
		suite.uploadFilesForTest(ctx, filepath, filepath)
	}

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

	expected := adapterentities.ObjectScanResponse{
		Result: map[string]adapterentities.ScanResponse{
			suite.bucketName: {
				Bucket:   suite.bucketName,
				Scanned:  39,
				Bypassed: 3,
				Matches:  0,
				Errors:   0,
				Entropy:  entities.GenerateEntropyBuckets([9]int{1, 0, 0, 1, 3, 24, 5, 3, 2}),
				Requests: 16,
			},
		},
	}

	request, _ := http.NewRequest("GET", fmt.Sprintf("http://localhost:3000/v1/objects?bucket=%s", suite.bucketName), http.NoBody)
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	client := &http.Client{}

	suite.Require().Eventually(func() bool {
		httpResponse, err := client.Do(request)
		if err != nil || httpResponse.StatusCode != http.StatusOK {
			return false
		}

		var obtained adapterentities.ObjectScanResponse
		body, _ := io.ReadAll(httpResponse.Body)
		json.Unmarshal(body, &obtained)
		if expected.Error != "" {
			return false
		}

		return reflect.DeepEqual(expected, obtained)
	}, 2*time.Minute, 10*time.Second)
}
