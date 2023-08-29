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
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"eagle-eye/common"
	"eagle-eye/domain/ports/out"
	"eagle-eye/fileutils"
	"eagle-eye/logging"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/pierrec/lz4/v4"
	"github.com/spf13/afero"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

const baseDir = ""

type DecompressService struct {
	logger logging.Logger
}

func NewDecompressService(logger logging.Logger) DecompressService {
	return DecompressService{logger: logger}
}

func (d *DecompressService) Extract(storage out.LocalStorage, buffer []byte) error {
	walkDecompress, wasDecompressed := getDecompressFunctions(storage, buffer)

	err := afero.Walk(storage, baseDir, walkDecompress)

	if err != nil {
		return err
	}

	if !wasDecompressed() {
		return nil
	}

	return d.Extract(storage, buffer)
}

func getDecompressFunctions(storage out.LocalStorage, buffer []byte) (walkFn filepath.WalkFunc, wasDecompressed func() bool) {
	hasDecompressed := false

	return func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := storage.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file during extraction")
			}
			defer file.Close()

			compressedType, err := fileutils.GetCompressedType(file)

			switch {
			case errors.Is(err, fileutils.ErrCantReadHeader):
				return fmt.Errorf("failed to read header")
			case errors.Is(err, fileutils.ErrUnknownCompressedType):
				return nil
			}

			if err := extract(compressedType, path, storage, buffer); err != nil {
				return fmt.Errorf("failed to extract file")
			}

			if err := storage.Remove(path); err != nil {
				return fmt.Errorf("failed to remove extracted file")
			}

			hasDecompressed = true

			return nil
		},
		func() bool {
			return hasDecompressed
		}
}

func extract(compressedType fileutils.CompressedType, filename string, storage out.LocalStorage, buffer []byte) error {
	switch compressedType {
	case fileutils.Gzfile:
		return extractGz(filename, storage, buffer)

	case fileutils.Tarfile:
		return extractTar(filename, storage, buffer)

	case fileutils.Zipfile:
		return extractZip(filename, storage, buffer)

	case fileutils.Lz4file:
		return extractLz4(filename, storage, buffer)

	case fileutils.Gitbundle:
		return extractGitBundle(filename, storage)

	default:
		return fmt.Errorf("unsupported compressed type")
	}
}

func extractGitBundle(filename string, storage out.LocalStorage) error {
	tmpDir := fmt.Sprintf("/tmp/%s", uuid.New())
	defer os.RemoveAll(tmpDir)

	err := storage.DumpToDisk(tmpDir)
	if err != nil {
		return err
	}

	err = storage.Destroy()
	if err != nil {
		return err
	}

	fullpath := common.GeneratedExtractedFilename(fmt.Sprintf("%s/%s", tmpDir, filename), []string{"bundle"})

	// git only operates on disk, so we need a compatibility layer to write to disk and then read back to our filesystem
	var stdoutBuffer bytes.Buffer
	cmd := exec.Command("git", "clone", fmt.Sprintf("%s/%s", tmpDir, filename), fullpath)
	cmd.Stdout = &stdoutBuffer

	if err = cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repo from git bundle file: %w", err)
	}

	// Load to disk or memory as required
	return storage.RestoreFromDisk(tmpDir)
}

func extractZip(filename string, storage out.LocalStorage, buffer []byte) error {
	file, err := storage.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	fileSize, err := storage.Size(filename)
	if err != nil {
		return err
	}

	r, err := zip.NewReader(file, fileSize)
	if err != nil {
		return err
	}

	dir := common.GeneratedExtractedFilename(filename, []string{"zip"})

	for _, file := range r.File {
		if file.FileInfo().IsDir() {
			continue
		}

		if err := extractSingleZipFile(file, fmt.Sprintf("%s/%s", dir, file.FileInfo().Name()), storage, buffer); err != nil {
			return err
		}
	}

	return nil
}

func extractSingleZipFile(file *zip.File, fullpath string, storage out.LocalStorage, buffer []byte) error {
	outFile, err := storage.Create(fullpath)
	if err != nil {
		return fmt.Errorf("failed to create file for extration: %w", err)
	}
	defer outFile.Close()

	zipReader, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file for extraction: %w", err)
	}
	defer zipReader.Close()

	// Zip suffers from the same behavior as gz. Only 32Kb are read at a time.
	for {
		written, err := zipReader.Read(buffer)
		eof := errors.Is(err, io.EOF)

		if !eof && err != nil {
			return fmt.Errorf("failed to read bytes during extraction: %w", err)
		}

		_, err = outFile.Write(buffer[:written])
		if err != nil {
			return fmt.Errorf("failed to write bytes: %w", err)
		}

		if eof {
			break
		}

		if err != nil {
			return fmt.Errorf("failed to copy bytes during extraction: %w", err)
		}
	}

	return nil
}

func extractTar(filename string, storage out.LocalStorage, buffer []byte) error {
	file, err := storage.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	tarReader := tar.NewReader(file)

	dir := common.GeneratedExtractedFilename(filename, []string{"tar"})

	// TAR format is not good for random access.
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return err
		}

		if header.Typeflag == tar.TypeReg {
			if err := extractSingleTARFile(tarReader, fmt.Sprintf("%s/%s", dir, header.Name), storage, buffer); err != nil {
				return err
			}
		}
	}

	return nil
}

func extractSingleTARFile(tarReader *tar.Reader, fullpath string, storage out.LocalStorage, buffer []byte) error {
	outFile, err := storage.Create(fullpath)
	if err != nil {
		return fmt.Errorf("failed to create file for extration: %w", err)
	}
	defer outFile.Close()

	// Although the tar format doesn't suffer from the same issue as
	// the gz format, we'll extract the file completely as long as we have memory available.
	for {
		read, err := tarReader.Read(buffer)
		eof := errors.Is(err, io.EOF)

		if !eof && err != nil {
			return fmt.Errorf("failed to read bytes during extraction: %w", err)
		}

		if _, err := outFile.Write(buffer[:read]); err != nil {
			return fmt.Errorf("failed to write bytes during extraction: %w", err)
		}

		if eof {
			break
		}

		if err != nil {
			return fmt.Errorf("failed to copy bytes during extraction: %w", err)
		}
	}

	return nil
}

func extractLz4(filename string, storage out.LocalStorage, buffer []byte) error {
	// Lz4 by itself does not support multiple files.
	file, err := storage.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	lzReader := lz4.NewReader(file)
	fullpath := common.GeneratedExtractedFilename(filename, []string{"lz4"})

	outFile, err := storage.Create(fullpath)
	if err != nil {
		return fmt.Errorf("failed to create file for extration: %w", err)
	}
	defer outFile.Close()

	for {
		// Although the buffer has a bigger size, gzread.read limits it reads to 32Kb and that's why we are
		// performing a loop in here.
		written, err := lzReader.Read(buffer)
		eof := errors.Is(err, io.EOF)

		if !eof && err != nil {
			return fmt.Errorf("failed to read bytes during extraction: %w", err)
		}

		_, err = outFile.Write(buffer[:written])
		if err != nil {
			return fmt.Errorf("failed to write bytes during extraction: %w", err)
		}

		if eof {
			break
		}

		if err != nil {
			return fmt.Errorf("failed to copy bytes during extraction: %w", err)
		}
	}

	return nil
}

func extractGz(filename string, storage out.LocalStorage, buffer []byte) error {
	file, err := storage.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// GZ by itself does not support multiple files.
	gzRead, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzRead.Close()

	fullpath := common.GeneratedExtractedFilename(filename, []string{"gz", "tgz"})
	outFile, err := storage.Create(fullpath)

	if err != nil {
		return fmt.Errorf("failed to create file for extration: %w", err)
	}
	defer outFile.Close()

	for {
		// Although the buffer has a bigger size, gzread.read limits it reads to 32Kb and that's why we are
		// performing a loop in here.
		written, err := gzRead.Read(buffer)
		eof := errors.Is(err, io.EOF)

		if !eof && err != nil {
			return fmt.Errorf("failed to read bytes during extraction: %w", err)
		}

		_, err = outFile.Write(buffer[:written])
		if err != nil {
			return fmt.Errorf("failed to write bytes during extraction: %w", err)
		}

		if eof {
			break
		}
	}

	return nil
}
