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

package services

import (
	"eagle-eye/adapters/out"
	"eagle-eye/common"
	"eagle-eye/logging"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractFiles(t *testing.T) {
	type test struct {
		filename string
		files    int
	}

	tests := []test{
		{filename: "text.gz", files: 1},
		{filename: "text.txt.gz", files: 1},
		{filename: "text.zip", files: 1},
		{filename: "text.lz4", files: 1},
		// Be sure your git version >= 2.42.0
		{filename: "repo.bundle", files: 29},
		{filename: "nested.zip", files: 31},
	}

	d := NewDecompressService(logging.NewDiscardLog())
	localStorageFactory := out.NewLocalStorageFactory(1024 * 1024 * 1024)

	for _, tc := range tests {
		tc := tc
		t.Run(tc.filename, func(t *testing.T) {
			storage, _ := localStorageFactory.GetLocalStorage(0, false)

			file, err := storage.Create(tc.filename)
			assert.NoError(t, err)

			common.LoadFileToStorage(t, tc.filename, file)

			err = d.Extract(storage, make([]byte, 1024*1024))
			assert.NoError(t, err)

			files, err := storage.ListFiles("")

			assert.NoError(t, err)
			assert.Equal(t, tc.files, len(files), "list of files %+v", files)
		})
	}
}
