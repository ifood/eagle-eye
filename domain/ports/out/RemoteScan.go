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

import "context"

type QueryResult int8

const (
	Error QueryResult = iota
	Benign
	Malicious
	Unseen
	InProgress
	DecodeError
	InvalidID
)

type QueryStatus struct {
	ID             string
	AnalysisResult QueryResult
	Error          error
}

/*
Currently, we integrate only with VirusTotal external scan.
Assumptions:
- (i) Rate limit may not be enough to scan all binaries
- (ii) Binary files won't be found in the scanner database
- (iii) Binary scan endpoint merely queries the scan
- (iv) User may want to not use. (eg.: by not passing an apikey)

Therefore:
- (i) we will track the usage through a global rate limiter
- (ii) we won't query by hash
- (iii) scanning a binary will require 2 method calls, first call
to schedule the scanning and second one to obtain the result
*/
type RemoteScan interface {
	IsAvailable() bool
	ScanHash(ctx context.Context, hash string) QueryStatus
	ScanBinary(ctx context.Context, data []byte) QueryStatus
	GetScanResult(ctx context.Context, ID string) QueryStatus
}
