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
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"math"
)

type DecryptionEngine struct {
	cbc       cipher.BlockMode
	lastBlock []byte
}

func NewDecryptionEngine(passphrase string, salt []byte) (*DecryptionEngine, error) {
	params := generateKeyFromPassword([]byte(passphrase), salt)

	c, err := aes.NewCipher(params.key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cipher with parameters. %w", err)
	}

	return &DecryptionEngine{cbc: cipher.NewCBCDecrypter(c, params.iv)}, nil
}

func (d *DecryptionEngine) DecryptBlock(data []byte) ([]byte, error) {
	d.lastBlock = append(d.lastBlock, data...)

	availableBlocks := aes.BlockSize * ((len(d.lastBlock) - 1) / aes.BlockSize)
	size := int(math.Max(float64(availableBlocks), 0))

	decrypted := make([]byte, size)
	if size != 0 {
		d.cbc.CryptBlocks(decrypted, d.lastBlock[:size])
		d.lastBlock = d.lastBlock[size:]
	}

	return decrypted, nil
}

func (d *DecryptionEngine) DecryptEnd() ([]byte, error) {
	decrypted := make([]byte, len(d.lastBlock))
	d.cbc.CryptBlocks(decrypted, d.lastBlock)

	return pkcs7Unpad(decrypted, aes.BlockSize)
}
