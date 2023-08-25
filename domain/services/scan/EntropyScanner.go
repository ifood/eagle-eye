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

package scan

import (
	"context"
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"errors"
	"io"
	"math"
	"strconv"
)

type EntropyScanner struct {
	logger logging.Logger
}

func NewEntropyScanner(logger logging.Logger) *EntropyScanner {
	return &EntropyScanner{logger: logger}
}

func (e *EntropyScanner) Scan(ctx context.Context, sc scanContext) (entities.ScanResult, error) {
	size := 0
	entropy := 0.0
	const rangeOfByteValue = 256
	byteCounts := make([]int, rangeOfByteValue)

	result := entities.ScanResult{Entropy: entities.GenerateEntropyBuckets([9]int{})}

	file, err := sc.Storage.Open(sc.Filename)
	if err != nil {
		e.logger.Errorw("couldn't open file to calculate entropy", "error", err, "filename", sc.Filename)
		return result, err
	}
	defer file.Close()

	for {
		// Read does get all the possible bytes until the size of the buffer
		numBytesRead, err := file.Read(sc.Buffer)
		size += numBytesRead
		// For each byte of the data that was read, increment the count
		// of that number of bytes seen in the file in our byteCounts
		// array
		for i := 0; i < numBytesRead; i++ {
			byteCounts[int(sc.Buffer[i])]++
		}

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			e.logger.Errorw("failed to read file during entropy calculation", "error", err, "file", "filename")
			result.Errors++

			return result, err
		}
	}

	if size == 0 {
		result.Entropy["0"]++
		return result, nil
	}

	for i := 0; i < rangeOfByteValue; i++ {
		px := float64(byteCounts[i]) / float64(size)
		if px > 0 {
			entropy += -px * math.Log2(px)
		}
	}

	result.Entropy[strconv.Itoa(int(math.Ceil(entropy)))]++

	return result, nil
}
