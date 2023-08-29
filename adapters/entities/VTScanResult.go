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

type VTScanResult struct {
	Data Data
}

type Data struct {
	Attributes Attributes `json:"attributes"`
	ID         string     `json:"id"`
}

type Attributes struct {
	Status string `json:"status"`
	// LastAnalysisStats filled when consulting existing hash
	LastAnalysisStats AnalysisStats `json:"last_analysis_stats"`
	// Stats filled when uploading file
	Stats AnalysisStats `json:"stats"`
}

type AnalysisStats struct {
	Malicious  int `json:"malicious"`
	Undetected int `json:"undetected"`
}
