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

package crypto

import (
	"encoding/base64"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDecryptSingleCall(t *testing.T) {
	d, err := NewDecryptionEngine("passphrase", []byte("12345678"))
	assert.NoError(t, err)

	data, _ := base64.StdEncoding.DecodeString("V1xFqK8IMvw+SpPDEZYan6W+50DS4RTsMe9zHW4xAcc=")
	blocks, err := d.DecryptBlock(data)
	assert.NoError(t, err)

	_, err = d.DecryptEnd()
	assert.NoError(t, err)
	assert.Equal(t, "1234567891234567", string(blocks))
}

func TestDecryptMultipleCall(t *testing.T) {
	d, err := NewDecryptionEngine("passphrase", []byte("12345678"))
	assert.NoError(t, err)

	block := make([]byte, 0)

	data, _ := base64.StdEncoding.DecodeString("V1xFqK8IMvw+SpPDEZYan6W+50DS4RTsMe9zHW4xAcc=")
	for _, singleByte := range data {
		encByte, err := d.DecryptBlock([]byte{singleByte})
		assert.NoError(t, err)

		block = append(block, encByte...)
	}

	assert.Equal(t, "1234567891234567", string(block))

	endBlock, err := d.DecryptEnd()
	assert.NoError(t, err)
	assert.Equal(t, "", string(endBlock))
}
