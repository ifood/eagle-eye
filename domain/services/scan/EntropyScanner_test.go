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
	adapter "eagle-eye/adapters/out"
	"eagle-eye/common"
	"eagle-eye/logging"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

const maxsizeBuffer = 1024 * 1024
const maxStorageUsage = 1024 * 1024 * 50

func TestMultipleFiles(t *testing.T) {
	localStorageFactory := adapter.NewLocalStorageFactory(maxStorageUsage)

	memStorage, _ := localStorageFactory.GetLocalStorage(0, false)
	type test struct {
		filename      string
		entropyBucket string
	}

	tests := []test{
		{filename: "text.txt", entropyBucket: "5"},
		{filename: "cabeca_batata.jpeg", entropyBucket: "8"},
		{filename: "monke.png", entropyBucket: "8"},
		{filename: "pepe_hack.gif", entropyBucket: "8"},
		{filename: "emptyfile", entropyBucket: "0"},
		{filename: "text.gz", entropyBucket: "8"},
		{filename: "text.txt.gz", entropyBucket: "8"},
		{filename: "text.zip", entropyBucket: "8"},
		{filename: "text.lz4", entropyBucket: "7"},
		{filename: "repo.bundle", entropyBucket: "8"},
	}

	entropyScanner := NewEntropyScanner(logging.NewDiscardLog())

	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("Checking entropy of file %s", tc.filename), func(t *testing.T) {
			file, _ := memStorage.Create(tc.filename)
			common.LoadFileToStorage(t, tc.filename, file)
			file.Close()

			sc := scanContext{Buffer: make([]byte, maxsizeBuffer), Filename: tc.filename, Storage: memStorage}
			result, err := entropyScanner.Scan(context.Background(), sc)
			assert.NoError(t, err)
			assert.NotZero(t, result.Entropy[tc.entropyBucket])
		})
	}
}
