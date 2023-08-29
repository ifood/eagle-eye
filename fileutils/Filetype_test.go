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

package fileutils

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestImageTypes(t *testing.T) {
	table := []struct {
		name         string
		fileBytes    []byte
		expectedType Filetype
	}{
		{name: "bmp", fileBytes: []byte{0x42, 0x4d}, expectedType: Multimedia},
		{name: "jpg", fileBytes: []byte{0xff, 0xd8, 0xff, 0xe0}, expectedType: Multimedia},
		{name: "png", fileBytes: []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, expectedType: Multimedia},
		{name: "gif87a", fileBytes: []byte{0x47, 0x49, 0x46, 0x38, 0x37, 0x61}, expectedType: Multimedia},
		{name: "gif89a", fileBytes: []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}, expectedType: Multimedia},
	}

	for _, v := range table {
		v := v
		t.Run(v.name, func(t *testing.T) {
			actualType, err := GetType(bytes.NewReader(v.fileBytes))
			assert.NoError(t, err)
			assert.Equal(t, v.expectedType, actualType)
		})
	}
}

func TestCompressedTypes(t *testing.T) {
	tarbytes := make([]byte, 512)
	copy(tarbytes[257:], []byte{0x75, 0x73, 0x74, 0x61, 0x72})

	table := []struct {
		name         string
		fileBytes    []byte
		expectedType CompressedType
	}{
		{name: "zipfile", fileBytes: []byte{0x50, 0x4B, 0x03, 0x04}, expectedType: Zipfile},
		{name: "gzfile", fileBytes: []byte{0x1f, 0x8b}, expectedType: Gzfile},
		{name: "lz4file", fileBytes: []byte{0x04, 0x22, 0x4D, 0x18}, expectedType: Lz4file},
		{name: "gitbundle", fileBytes: []byte("# v2 git bundle"), expectedType: Gitbundle},
		{name: "tarfile", fileBytes: tarbytes, expectedType: Tarfile},
	}

	for _, v := range table {
		v := v
		t.Run(v.name, func(t *testing.T) {
			actualType, err := GetCompressedType(bytes.NewReader(v.fileBytes))
			assert.NoError(t, err)
			assert.Equal(t, v.expectedType, actualType)
		})
	}
}

func TestNonExecutables(t *testing.T) {
	tarbytes := make([]byte, 512)
	copy(tarbytes[257:], []byte{0x75, 0x73, 0x74, 0x61, 0x72})

	table := []struct {
		name         string
		fileBytes    []byte
		expectedType Filetype
	}{
		{name: "uncompressed", fileBytes: []byte{0xde, 0xad, 0xbe, 0xef}, expectedType: Uncompressed},
		{name: "zipfile", fileBytes: []byte{0x50, 0x4B, 0x03, 0x04}, expectedType: Compressed},
		{name: "gzfile", fileBytes: []byte{0x1f, 0x8b}, expectedType: Compressed},
		{name: "lz4file", fileBytes: []byte{0x04, 0x22, 0x4D, 0x18}, expectedType: Compressed},
		{name: "gitbundle", fileBytes: []byte("# v2 git bundle"), expectedType: Compressed},
		{name: "tarfile", fileBytes: tarbytes, expectedType: Compressed},
	}

	for _, v := range table {
		v := v
		t.Run(v.name, func(t *testing.T) {
			actualType, err := GetType(bytes.NewReader(v.fileBytes))
			assert.NoError(t, err)
			assert.Equal(t, v.expectedType, actualType)
		})
	}
}

func TestEmptyFile(t *testing.T) {
	actualType, err := GetType(bytes.NewReader([]byte{}))
	assert.NoError(t, err)
	assert.Equal(t, Uncompressed, actualType)
}

func TestExecutables(t *testing.T) {
	table := []struct {
		name       string
		fileBytes  []byte
		executable bool
	}{
		{name: "eicar sample", fileBytes: []byte("X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"), executable: true},
		{name: "windows executable", fileBytes: []byte{0x4d, 0x5a}, executable: true},
		{name: "linux executable", fileBytes: []byte{0x7f, 0x45, 0x4c, 0x46, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, executable: true},
		{name: "Mach-O 32 bit", fileBytes: []byte{0xce, 0xfa, 0xed, 0xfe}, executable: true},
		{name: "Mach-O 64 bit", fileBytes: []byte{0xcf, 0xfa, 0xed, 0xfe}, executable: true},
		{name: "not an executable", fileBytes: []byte("not an executable"), executable: false},
	}

	for _, v := range table {
		v := v
		t.Run(v.name, func(t *testing.T) {
			assert.Equal(t, v.executable, IsExecutable(bytes.NewReader(v.fileBytes)))
		})
	}
}
