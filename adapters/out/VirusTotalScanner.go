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
	"bytes"
	"context"
	"eagle-eye/adapters/entities"
	"eagle-eye/common"
	"eagle-eye/domain/ports/out"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
)
import "net/http"

const (
	maxDetectionPercentage = 100.0
	smallFileLimit         = 32 * 1024 * 1024
	hardFileLimit          = 500 * 1024 * 1024
	scanHashURL            = "https://www.virustotal.com/api/v3/files"
)

type VirusTotalScanner struct {
	apiKey             string
	detectionThreshold float64
	rateLimiter        common.RateLimiter
}

func NewVirusTotalScanner(apiKey string, detectionThreshold float64, rateLimiter common.RateLimiter) *VirusTotalScanner {
	return &VirusTotalScanner{apiKey: apiKey, detectionThreshold: detectionThreshold, rateLimiter: rateLimiter}
}

func (v *VirusTotalScanner) IsAvailable() bool {
	return v.apiKey != ""
}

func (v *VirusTotalScanner) ScanHash(ctx context.Context, hash string) out.QueryStatus {
	if !v.rateLimiter.IsRequestAllowed() {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("too many requests")}
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/%s", scanHashURL, hash), http.NoBody)
	if err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("failed to encode request for virustotal. %w", err)}
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-apikey", v.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("request to virustotal failed. %w", err)}
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("http call failed with code %d", res.StatusCode)}
	}

	var result entities.VTScanResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.DecodeError, Error: fmt.Errorf("failed to decode url response. err: %w, body: %v", err, res.Body)}
	}

	detectionRate := maxDetectionPercentage * float64(result.Data.Attributes.LastAnalysisStats.Malicious) / float64(result.Data.Attributes.LastAnalysisStats.Malicious+result.Data.Attributes.LastAnalysisStats.Undetected+1)
	if detectionRate > v.detectionThreshold {
		return out.QueryStatus{ID: result.Data.ID, AnalysisResult: out.Malicious}
	}

	return out.QueryStatus{ID: result.Data.ID, AnalysisResult: out.Benign}
}

func (v *VirusTotalScanner) getUploadURL(ctx context.Context, filesize int) (url string, err error) {
	if filesize < smallFileLimit {
		return "https://www.virustotal.com/api/v3/files", nil
	}

	if filesize > hardFileLimit {
		return "", fmt.Errorf("above max entity size")
	}

	if !v.rateLimiter.IsRequestAllowed() {
		return "", fmt.Errorf("too many requests")
	}

	largeFilesURL := "https://www.virustotal.com/api/v3/files/upload_url"

	req, err := http.NewRequestWithContext(ctx, "get", largeFilesURL, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to encode request for virustotal. %w", err)
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-apikey", v.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to obtain url. %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error during get url call. http status %d", res.StatusCode)
	}

	vtURL := struct {
		Data string `json:"data"`
	}{}

	if err := json.NewDecoder(res.Body).Decode(&vtURL); err != nil {
		return "", fmt.Errorf("failed to decode url response. err: %w, body: %v", err, res.Body)
	}

	return vtURL.Data, nil
}

func (v *VirusTotalScanner) ScanBinary(ctx context.Context, data []byte) out.QueryStatus {
	url, err := v.getUploadURL(ctx, len(data))
	if err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: err}
	}

	if !v.rateLimiter.IsRequestAllowed() {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("too many requests")}
	}

	bodyRequest := new(bytes.Buffer)
	writer := multipart.NewWriter(bodyRequest)
	part, err := writer.CreateFormFile("file", "filename")

	if err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("failed to encode body request for virustotal. %w", err)}
	}

	_, err = io.Copy(part, bytes.NewReader(data))
	if err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("failed to prepare body request for virustotal. %w", err)}
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyRequest)
	if err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("failed to encode request for virustotal. %w", err)}
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-apikey", v.apiKey)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("request to virustotal failed. %w", err)}
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("request to virustotal failed with status %v", res.StatusCode)}
	}

	var result entities.VTScanResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.DecodeError, Error: fmt.Errorf("failed to decode url response. err: %w, body: %v", err, res.Body)}
	}

	// Sometimes, for whatever reason, VirusTotal returns a code which isn't a base64 code
	// Although the API returns 200 and the id for the scan, the id is invalid :/
	_, err = base64.StdEncoding.DecodeString(result.Data.ID)
	if err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.InvalidID, Error: fmt.Errorf("virustotal returned an invalid id. err: %w, id: %s", err, result.Data.ID)}
	}

	return out.QueryStatus{ID: result.Data.ID, AnalysisResult: out.InProgress}
}

func (v *VirusTotalScanner) GetScanResult(ctx context.Context, id string) out.QueryStatus {
	if !v.rateLimiter.IsRequestAllowed() {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("too many requests for virustotal")}
	}

	url := "https://www.virustotal.com/api/v3/analyses/%s"

	req, err := http.NewRequestWithContext(ctx, "get", fmt.Sprintf(url, id), http.NoBody)
	if err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("failed to encode request for virustotal. %w", err)}
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("x-apikey", v.apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("request to virustotal failed. %w", err)}
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return out.QueryStatus{ID: "", AnalysisResult: out.Error, Error: fmt.Errorf("request to virustotal failed with status %v", res.StatusCode)}
	}

	var result entities.VTScanResult
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return out.QueryStatus{ID: "", AnalysisResult: out.DecodeError, Error: fmt.Errorf("failed to decode url response. err: %w, body: %v", err, res.Body)}
	}

	if result.Data.Attributes.Status == "queued" {
		return out.QueryStatus{ID: result.Data.ID, AnalysisResult: out.InProgress}
	}

	detectionRate := maxDetectionPercentage * float64(result.Data.Attributes.Stats.Malicious) / float64(result.Data.Attributes.Stats.Malicious+result.Data.Attributes.Stats.Undetected+1)
	if detectionRate > v.detectionThreshold {
		return out.QueryStatus{ID: result.Data.ID, AnalysisResult: out.Malicious}
	}

	return out.QueryStatus{ID: result.Data.ID, AnalysisResult: out.Benign}
}
