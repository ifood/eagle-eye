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
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"fmt"
)

const keyAndIvSize = 48

type OpenSSLCreds struct {
	key []byte
	iv  []byte
}

func pkcs7Unpad(data []byte, blocklen int) ([]byte, error) {
	if blocklen <= 0 {
		return nil, fmt.Errorf("invalid blocklen %d", blocklen)
	}

	if len(data)%blocklen != 0 || len(data) == 0 {
		return nil, fmt.Errorf("invalid data len %d", len(data))
	}

	padlen := int(data[len(data)-1])
	if padlen > blocklen || padlen == 0 {
		return nil, fmt.Errorf("invalid padding")
	}

	pad := data[len(data)-padlen:]
	for i := 0; i < padlen; i++ {
		if pad[i] != byte(padlen) {
			return nil, fmt.Errorf("invalid padding")
		}
	}

	return data[:len(data)-padlen], nil
}

func pkcs7Pad(b []byte, blocklen int) ([]byte, error) {
	if blocklen <= 0 {
		return nil, fmt.Errorf("invalid blocklen %d", blocklen)
	}

	if len(b) == 0 {
		return nil, fmt.Errorf("invalid PKCS7 data (empty or not padded)")
	}

	n := blocklen - (len(b) % blocklen)
	pb := make([]byte, len(b)+n)
	copy(pb, b)
	copy(pb[len(b):], bytes.Repeat([]byte{byte(n)}, n))

	return pb, nil
}

func generateKeyFromPassword(password, salt []byte) OpenSSLCreds {
	m := make([]byte, keyAndIvSize)
	var prev []byte

	for i := 0; i < 3; i++ {
		prev = hash(prev, password, salt)
		copy(m[i*20:], prev)
	}

	return OpenSSLCreds{key: m[:32], iv: m[32:]}
}

func hash(prev, password, salt []byte) []byte {
	a := make([]byte, len(prev)+len(password)+len(salt))
	copy(a, prev)
	copy(a[len(prev):], password)
	copy(a[len(prev)+len(password):], salt)

	return sha1sum(a)
}

func sha1sum(data []byte) []byte {
	h := sha1.New()
	h.Write(data)

	return h.Sum(nil)
}

func Decrypt(passphrase string, data []byte) ([]byte, error) {
	if len(data) < aes.BlockSize {
		return nil, fmt.Errorf("Data too small for decription")
	}

	// Check starts with Salted
	saltHeader := data[:aes.BlockSize]
	salt := saltHeader[8:]
	creds := generateKeyFromPassword([]byte(passphrase), salt)

	return decrypt(creds.key, creds.iv, data)
}

func decrypt(key, iv, data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("bad blocksize(%v), aes.BlockSize = %v", len(data), aes.BlockSize)
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	cbc := cipher.NewCBCDecrypter(c, iv)
	cbc.CryptBlocks(data[aes.BlockSize:], data[aes.BlockSize:])
	out, err := pkcs7Unpad(data[aes.BlockSize:], aes.BlockSize)

	return out, err
}
