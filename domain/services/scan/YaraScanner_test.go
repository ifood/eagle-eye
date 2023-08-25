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
	adapters "eagle-eye/adapters/out"
	"eagle-eye/common"
	"eagle-eye/logging"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestYaraMatch(t *testing.T) {
	common.ChangePathForTesting(t)
	yaraScanner, err := NewYaraScanner("resources/testrules/", logging.NewDiscardLog())
	assert.NoError(t, err)

	localStorageFactory := adapters.NewLocalStorageFactory(1024 * 1024 * 10)
	memStorage, _ := localStorageFactory.GetLocalStorage(0, false)

	type test struct {
		testname string
		filename string
		content  string
		matches  int
	}

	tests := []test{
		{testname: "detect ransom", filename: "testransom", content: "IAMAMALWARE", matches: 1},
		{testname: "benign scan", filename: "testbenign", content: "BENIGN", matches: 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.testname, func(t *testing.T) {
			file, _ := memStorage.Create(tc.filename)
			_, err := file.WriteString(tc.content)
			assert.NoError(t, err)
			file.Close()

			sc := scanContext{Buffer: make([]byte, 1024*1024), Filename: tc.filename, Storage: memStorage}
			result, err := yaraScanner.Scan(context.Background(), sc)
			assert.NoError(t, err)
			assert.Equal(t, tc.matches, result.Matches)
		})
	}
}
