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

type EncryptionEngine struct {
	cbc       cipher.BlockMode
	lastBlock []byte
}

func NewEncryptionEngine(passphrase string, salt []byte) (*EncryptionEngine, error) {
	encEngine := &EncryptionEngine{}
	params := generateKeyFromPassword([]byte(passphrase), salt)

	c, err := aes.NewCipher(params.key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cipher with parameters. %w", err)
	}

	encEngine.cbc = cipher.NewCBCEncrypter(c, params.iv)

	return encEngine, nil
}

func (e *EncryptionEngine) EncryptBlock(data []byte) ([]byte, error) {
	e.lastBlock = append(e.lastBlock, data...)

	availableBlocks := aes.BlockSize * ((len(e.lastBlock) - 1) / aes.BlockSize)
	size := int(math.Max(float64(availableBlocks), 0))

	encrypted := make([]byte, size)
	if size != 0 {
		e.cbc.CryptBlocks(encrypted, e.lastBlock[:size])
		e.lastBlock = e.lastBlock[size:]
	}

	return encrypted, nil
}

func (e *EncryptionEngine) EncryptEnd() ([]byte, error) {
	encrypted, _ := pkcs7Pad(e.lastBlock, aes.BlockSize)
	e.cbc.CryptBlocks(encrypted, encrypted)

	return encrypted, nil
}
