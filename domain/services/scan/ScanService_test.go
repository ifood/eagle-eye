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
	adaptersout "eagle-eye/adapters/out"
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestScanService(t *testing.T) {
	localStorageFactory := adaptersout.NewLocalStorageFactory(1024 * 1024 * 10)

	type test struct {
		filename string
		scanned  int
		bypassed int
		errors   int
	}

	tests := []test{
		{filename: "text.txt", scanned: 1, bypassed: 0, errors: 0},
		{filename: "cabeca_batata.jpeg", scanned: 0, bypassed: 1, errors: 0},
		{filename: "monke.png", scanned: 0, bypassed: 1, errors: 0},
		{filename: "pepe_hack.gif", scanned: 0, bypassed: 1, errors: 0},
		{filename: "emptyfile", scanned: 1, bypassed: 0, errors: 0},
	}

	scanService := NewScanService(localStorageFactory, nil, nil, logging.NewDiscardLog())

	for _, tc := range tests {
		tc := tc
		t.Run(tc.filename, func(t *testing.T) {
			memStorage, err := localStorageFactory.GetLocalStorage(0, false)
			assert.NoError(t, err)

			file, err := memStorage.Create(tc.filename)
			assert.NoError(t, err)

			common.LoadFileToStorage(t, tc.filename, file)
			result := scanService.Scan(context.Background(), entities.ScanRequest{Key: []string{tc.filename}, StorageID: memStorage.GetID()})

			assert.Equal(t, tc.scanned, result.Scanned)
			assert.Equal(t, tc.bypassed, result.Bypassed)
			assert.Equal(t, tc.errors, result.Errors)
		})
	}
}
