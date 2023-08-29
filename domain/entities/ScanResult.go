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

import (
	"eagle-eye/common"
	"strconv"
	"time"
)

type ScanResult struct {
	ScanID     string
	ResultType ResultType
	Bucket     string
	Scanned    int
	Bypassed   int
	Matches    int
	Errors     int
	Entropy    map[string]int
	Requests   int
	LastUpdate time.Time
}

func GenerateEntropyBuckets(frequencies [9]int) map[string]int {
	entropyBucket := make(map[string]int)
	for i := 0; i <= 8; i++ {
		entropyBucket[strconv.Itoa(i)] = frequencies[i]
	}

	return entropyBucket
}

func NewScanResult(bucket string) ScanResult {
	return ScanResult{
		Bucket:     bucket,
		Scanned:    0,
		Matches:    0,
		Bypassed:   0,
		Errors:     0,
		Entropy:    GenerateEntropyBuckets([9]int{0, 0, 0, 0, 0, 0, 0, 0}),
		LastUpdate: time.Now(),
	}
}

func CombineEntropyMap(a, b map[string]int) map[string]int {
	if a == nil {
		return CombineEntropyMap(GenerateEntropyBuckets([9]int{0, 0, 0, 0, 0, 0, 0, 0}), b)
	}

	if b == nil {
		return CombineEntropyMap(a, GenerateEntropyBuckets([9]int{0, 0, 0, 0, 0, 0, 0, 0}))
	}

	return CombineMap(a, b)
}

func CombineMap(a, b map[string]int) map[string]int {
	res := make(map[string]int)

	for keyA, valueA := range a {
		tmpValue := valueA
		if valueB, ok := b[keyA]; ok {
			tmpValue += valueB
		}

		res[keyA] = tmpValue
	}

	return res
}

func MergeScanResults(a, b ScanResult) ScanResult {
	return ScanResult{
		Bucket:     common.GetFirstNonEmpty(a.Bucket, b.Bucket, "no bucket specified"),
		Scanned:    a.Scanned + b.Scanned,
		Bypassed:   a.Bypassed + b.Bypassed,
		Matches:    a.Matches + b.Matches,
		Errors:     a.Errors + b.Errors,
		Entropy:    CombineEntropyMap(a.Entropy, b.Entropy),
		Requests:   a.Requests + b.Requests,
		LastUpdate: common.GetMaxDate(a.LastUpdate, b.LastUpdate),
	}
}
