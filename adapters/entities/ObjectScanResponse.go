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

package entities

import "eagle-eye/domain/entities"

type ObjectScanResponse struct {
	Result map[string]ScanResponse `json:"result,omitempty"`
	Error  string                  `json:"error,omitempty"`
}

func MapResultToScanResponse(data map[string]entities.ScanResult) map[string]ScanResponse {
	result := make(map[string]ScanResponse)

	for key, value := range data {
		result[key] = MapToScanResponse(value)
	}

	return result
}
