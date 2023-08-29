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

package out

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/spf13/afero"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const defaultDirPermission = 0755

type LocalStorageFS struct {
	changeFSUsage func(storageID string, nbytes int64) error
	storageID     string
	afero.Fs
}

func NewLocalStorageFS(base afero.Fs, changeFSUsage func(storageID string, nbytes int64) error) (*LocalStorageFS, error) {
	// Enforcing base directory, because we don't want any file to escape the sandbox directory
	storageID := uuid.New().String()
	rootDir := "/tmp/" + storageID
	err := base.Mkdir(rootDir, defaultDirPermission)

	if err != nil {
		return nil, err
	}

	return &LocalStorageFS{changeFSUsage: changeFSUsage, storageID: storageID, Fs: afero.NewBasePathFs(base, rootDir)}, nil
}

func (d *LocalStorageFS) Create(path string) (afero.File, error) {
	if err := d.MkdirAll(filepath.Dir(path), defaultDirPermission); err != nil {
		return nil, err
	}

	file, err := d.Fs.Create(path)
	if err != nil {
		return nil, err
	}

	return NewLimitedFileSize(file, func(nbytes int64) error { return d.changeFSUsage(d.storageID, nbytes) }), nil
}

func (d *LocalStorageFS) DumpToDisk(target string) error {
	files, err := d.ListFiles("")
	if err != nil {
		return fmt.Errorf("dump to disk failed. %w", err)
	}

	for _, file := range files {
		err := func() error {
			srcFile, err := d.Open(file)
			if err != nil {
				return err
			}
			defer srcFile.Close()

			dir := filepath.Dir(file)
			err = os.MkdirAll(filepath.Join(target, dir), defaultDirPermission)

			if err != nil {
				return err
			}

			dstFile, err := os.Create(filepath.Join(target, file))
			if err != nil {
				return err
			}
			defer dstFile.Close()

			_, err = io.Copy(dstFile, srcFile)

			return err
		}()

		if err != nil {
			return fmt.Errorf("failed to dump to disk. %w", err)
		}
	}

	return nil
}

func (d *LocalStorageFS) RestoreFromDisk(src string) error {
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		storagePath := strings.TrimPrefix(path, src)
		if info.IsDir() {
			err = d.MkdirAll(storagePath, defaultDirPermission)
			if err != nil {
				return err
			}
		} else {
			srcFile, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file. %w", err)
			}
			defer srcFile.Close()

			dstFile, err := d.Create(storagePath)
			if err != nil {
				return fmt.Errorf("failed to create file on storage. %w", err)
			}
			defer dstFile.Close()

			_, err = io.Copy(dstFile, srcFile)

			return err
		}
		return nil
	})

	return err
}

func (d *LocalStorageFS) GetID() string {
	return d.storageID
}

func (d *LocalStorageFS) Destroy() error {
	return d.RemoveAll("")
}

func (d *LocalStorageFS) Exists(path string) (bool, error) {
	return afero.Exists(d.Fs, path)
}

func (d *LocalStorageFS) IsRegular(path string) (bool, error) {
	dir, err := afero.IsDir(d.Fs, path)
	return !dir, err
}

func (d *LocalStorageFS) Size(path string) (int64, error) {
	info, err := d.Stat(path)
	if err != nil {
		return 0, err
	}

	return info.Size(), err
}

func (d *LocalStorageFS) ListFiles(path string) ([]string, error) {
	fileList := make([]string, 0)

	err := afero.Walk(d.Fs, path, func(path string, f os.FileInfo, err error) error {
		if f.Mode().IsRegular() {
			fileList = append(fileList, path)
		}
		return err
	})

	return fileList, err
}
