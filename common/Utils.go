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

package common

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func RandFloat64() float64 {
	const arbitraryMaxValue = 2 << 32

	bigNum, err := rand.Int(rand.Reader, big.NewInt(arbitraryMaxValue))
	if err != nil {
		return 0
	}

	return float64(bigNum.Int64()) / float64(arbitraryMaxValue)
}

func RandInt(max int64) int64 {
	bigNum, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		log.Printf("Failed to generate rand int\n")
	}

	return bigNum.Int64()
}

func GeneratedExtractedFilename(filename string, extensions []string) string {
	// List of allowed extensions
	for _, extension := range extensions {
		if strings.HasSuffix(filename, fmt.Sprintf(".%s", extension)) {
			return strings.Split(filename, fmt.Sprintf(".%s", extension))[0]
		}
	}

	// File do not respect the extension, we should simply generate a random name for it.
	return uuid.New().String()
}

func GetMaxDate(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}

	return b
}

func ConvertNumberToHumanReadable(value int) string {
	const kilo = 1000
	const mega = 1000000
	const giga = 1000000000
	const tera = 1000000000000
	values := []float64{kilo, mega, giga, tera}
	prefixes := []string{"k", "M", "G", "T"}

	for index, limit := range values {
		if float64(value) < limit {
			index--
			if index < 0 {
				return fmt.Sprintf("%d", value)
			}

			return fmt.Sprintf("%.2f%s", float64(value)/values[index], prefixes[index])
		}
	}

	return fmt.Sprintf("%d", value)
}

func CreateEmptyEntropyBuckets() map[string]int {
	entropyBucket := make(map[string]int)
	for i := 0; i <= 8; i++ {
		entropyBucket[strconv.Itoa(i)] = 0
	}

	return entropyBucket
}

func GetFirstNonEmpty(a, b, defaultValue string) string {
	if a == "" && b == "" {
		return defaultValue
	}

	if a == "" {
		return b
	}

	return a
}

func ParseDate(date string, defaultTime time.Time) (time.Time, error) {
	if date == "" {
		return defaultTime, nil
	}

	return time.Parse("2006-01-02", date)
}

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}
