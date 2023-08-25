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
	"errors"
	"fmt"
	"github.com/gabriel-vasile/mimetype"
	"io"
	"strings"
	"sync"
)

type Filetype int8
type CompressedType int8

const (
	Zipfile CompressedType = iota + 1
	Tarfile
	Gzfile
	Lz4file
	Gitbundle
)

const (
	Uncompressed Filetype = iota + 1
	Executable
	Compressed
	Multimedia
)

const maxHeaderBuffer = 1024
const mimeApplicationType = "application"

var (
	ErrCantReadHeader        = errors.New("cant read file header")
	ErrUnknownCompressedType = errors.New("unknown compressed type")
)

//nolint:gochecknoglobals
var once sync.Once

func prefix(preffix []byte) func([]byte, uint32) bool {
	return func(raw []byte, limit uint32) bool {
		if limit < uint32(len(preffix)) {
			return false
		}

		return bytes.Equal(raw[:len(preffix)], preffix)
	}
}

func registerAdditionalTypes() {
	// Support for Eicar
	mimetype.Extend(prefix([]byte{0x58, 0x35, 0x4f, 0x21}), "application/x-eicar", "")

	// Support for Gitbundle
	mimetype.Extend(prefix([]byte("# v2 git bundle")), "application/x-gitbundle", "")

	// Support for LZ4
	mimetype.Extend(prefix([]byte{0x04, 0x22, 0x4D, 0x18}), "application/x-lz4", "")
}

func GetType(reader io.Reader) (Filetype, error) {
	once.Do(registerAdditionalTypes)
	return checkFiletype(reader)
}

func GetCompressedType(reader io.Reader) (CompressedType, error) {
	once.Do(registerAdditionalTypes)
	head := make([]byte, maxHeaderBuffer)
	_, err := reader.Read(head)

	if err != nil && !errors.Is(err, io.EOF) {
		return 0, ErrCantReadHeader
	}

	mtype := mimetype.Detect(head)
	identifiedType := strings.Split(mtype.String(), "/")

	switch {
	case isZip(identifiedType):
		return Zipfile, nil
	case isTar(identifiedType):
		return Tarfile, nil
	case isGzip(identifiedType):
		return Gzfile, nil
	case isGitbundle(identifiedType):
		return Gitbundle, nil
	case isLZ4(identifiedType):
		return Lz4file, nil
	default:
		return 0, ErrUnknownCompressedType
	}
}

func IsCompressed(filename string) bool {
	suffixes := []string{".tar", ".tar.gz", ".gz", ".zip", ".lz4", ".lz", "tgz"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(filename, suffix) {
			return true
		}
	}

	return false
}

//nolint:cyclop
func checkFiletype(reader io.Reader) (Filetype, error) {
	head := make([]byte, maxHeaderBuffer)
	_, err := reader.Read(head)

	if err != nil && !errors.Is(err, io.EOF) {
		return 0, fmt.Errorf("failed to read header from file. Error: %w", err)
	}

	mtype := mimetype.Detect(head)
	identifiedType := strings.Split(mtype.String(), "/")

	switch {
	case isMultimedia(identifiedType):
		return Multimedia, nil
	case isCompressed(identifiedType):
		return Compressed, nil
	case isBinaryApp(identifiedType):
		return Executable, nil
	default:
		return Uncompressed, nil
	}
}

func isCompressed(identifiedType []string) bool {
	return isZip(identifiedType) || isTar(identifiedType) || isGzip(identifiedType) || isGitbundle(identifiedType) || isLZ4(identifiedType)
}

func isLZ4(identifiedType []string) bool {
	return identifiedType[0] == mimeApplicationType &&
		identifiedType[1] == "x-lz4"
}

func isGitbundle(identifiedType []string) bool {
	return identifiedType[0] == mimeApplicationType &&
		identifiedType[1] == "x-gitbundle"
}

func isGzip(identifiedType []string) bool {
	return identifiedType[0] == mimeApplicationType &&
		identifiedType[1] == "gzip"
}

func isTar(identifiedType []string) bool {
	return identifiedType[0] == mimeApplicationType &&
		identifiedType[1] == "x-tar"
}

func isZip(identifiedType []string) bool {
	return identifiedType[0] == mimeApplicationType &&
		identifiedType[1] == "zip"
}

func isBinaryApp(identifiedType []string) bool {
	return identifiedType[0] == mimeApplicationType &&
		(identifiedType[1] == "x-elf" ||
			identifiedType[1] == "vnd.microsoft.portable-executable" ||
			identifiedType[1] == "x-executable" ||
			identifiedType[1] == "x-sharedlib" ||
			identifiedType[1] == "x-mach-binary" ||
			identifiedType[1] == "x-eicar")
}

func isMultimedia(identifiedType []string) bool {
	return identifiedType[0] == "audio" ||
		identifiedType[0] == "video" ||
		identifiedType[0] == "image"
}

func IsExecutable(reader io.Reader) bool {
	format, err := checkFiletype(reader)
	return err == nil && format == Executable
}
