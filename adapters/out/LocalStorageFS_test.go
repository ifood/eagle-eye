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
	"bytes"
	"eagle-eye/domain/ports/out"
	"fmt"
	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func TestDestroyMemStorage(t *testing.T) {
	localStorageFactory := NewLocalStorageFactory(1 * 1024 * 1024)

	memStorage, err := localStorageFactory.GetLocalStorage(1024, false)
	assert.NoError(t, err)

	_, err = localStorageFactory.GetStorageFromID(memStorage.GetID())
	assert.NoError(t, err)

	err = localStorageFactory.DestroyStorage(memStorage.GetID())
	assert.NoError(t, err)

	_, err = localStorageFactory.GetStorageFromID(memStorage.GetID())
	assert.Error(t, err)
}

func TestDestroyDiskStorage(t *testing.T) {
	localStorageFactory := NewLocalStorageFactory(5 * 1024 * 1024)

	diskStorage, err := localStorageFactory.GetLocalStorage(1024*1024*2, false)
	assert.NoError(t, err)

	_, err = localStorageFactory.GetStorageFromID(diskStorage.GetID())
	assert.NoError(t, err)

	err = localStorageFactory.DestroyStorage(diskStorage.GetID())
	assert.NoError(t, err)

	_, err = localStorageFactory.GetStorageFromID(diskStorage.GetID())
	assert.Error(t, err)
}

func TestReadWriteFile(t *testing.T) {
	changeStorageUsage := func(storageID string, nbytes int64) error { return nil }
	memStorage, err := NewLocalStorageFS(afero.NewMemMapFs(), changeStorageUsage)
	require.NoError(t, err)

	diskStorage, err := NewLocalStorageFS(afero.NewOsFs(), changeStorageUsage)
	require.NoError(t, err)

	table := []struct {
		storageType string
		storage     out.LocalStorage
	}{
		{storageType: "memory", storage: memStorage},
		{storageType: "disk", storage: diskStorage},
	}

	for _, v := range table {
		v := v
		t.Run(fmt.Sprintf("readwrite_%s", v.storageType), func(t *testing.T) {
			file, err := v.storage.Create("testfile")
			assert.NoError(t, err)

			expectedContext := "content"
			_, err = file.WriteString(expectedContext)
			assert.NoError(t, err)
			file.Close()

			file, err = v.storage.Open("testfile")
			assert.NoError(t, err)

			b := bytes.Buffer{}
			_, err = io.Copy(&b, file)
			assert.NoError(t, err)
			assert.Equal(t, []byte(expectedContext), b.Bytes())
		})
	}
}

func TestFileExists(t *testing.T) {
	changeStorageUsage := func(storageID string, nbytes int64) error { return nil }
	memStorage, _ := NewLocalStorageFS(afero.NewMemMapFs(), changeStorageUsage)
	diskStorage, _ := NewLocalStorageFS(afero.NewOsFs(), changeStorageUsage)

	table := []struct {
		storageType string
		storage     out.LocalStorage
	}{
		{storageType: "memory", storage: memStorage},
		{storageType: "disk", storage: diskStorage},
	}

	for _, v := range table {
		v := v
		t.Run(fmt.Sprintf("fileexist_%s", v.storageType), func(t *testing.T) {
			file, err := v.storage.Create("existfile")
			assert.NoError(t, err)
			file.Close()

			exists, err := v.storage.Exists("existfile")
			assert.True(t, exists)
			assert.NoError(t, err)

			exists, err = v.storage.Exists("notexist")
			assert.False(t, exists)
			assert.NoError(t, err)
		})
	}
}

func TestDeleteFile(t *testing.T) {
	changeStorageUsage := func(storageID string, nbytes int64) error { return nil }
	memStorage, _ := NewLocalStorageFS(afero.NewMemMapFs(), changeStorageUsage)
	diskStorage, _ := NewLocalStorageFS(afero.NewOsFs(), changeStorageUsage)

	table := []struct {
		storageType string
		storage     out.LocalStorage
	}{
		{storageType: "memory", storage: memStorage},
		{storageType: "disk", storage: diskStorage},
	}

	for _, v := range table {
		v := v
		t.Run(fmt.Sprintf("delete_%s", v.storageType), func(t *testing.T) {
			file, err := v.storage.Create("existfile")
			assert.NoError(t, err)
			file.Close()

			exists, err := v.storage.Exists("existfile")
			assert.True(t, exists)
			assert.NoError(t, err)

			err = v.storage.Remove("existfile")
			assert.NoError(t, err)

			exists, err = v.storage.Exists("existfile")
			assert.False(t, exists)
			assert.NoError(t, err)
		})
	}
}

func TestListFiles(t *testing.T) {
	changeStorageUsage := func(storageID string, nbytes int64) error { return nil }
	memStorage, err := NewLocalStorageFS(afero.NewMemMapFs(), changeStorageUsage)
	require.NoError(t, err)

	diskStorage, err := NewLocalStorageFS(afero.NewOsFs(), changeStorageUsage)
	require.NoError(t, err)

	table := []struct {
		storageType string
		storage     out.LocalStorage
	}{
		{storageType: "memory", storage: memStorage},
		{storageType: "disk", storage: diskStorage},
	}

	for _, v := range table {
		v := v
		t.Run(fmt.Sprintf("listfiles_%s", v.storageType), func(t *testing.T) {
			filenames := []string{"fileInRoot", "/folder1/file1", "/folder1/file2", "/folder2/file2", "/folder2/folder3/file"}
			for _, filename := range filenames {
				file, err := v.storage.Create(filename)
				assert.NoError(t, err)
				file.Close()
			}

			actualFilenames, err := v.storage.ListFiles("")
			assert.NoError(t, err)
			assert.Equal(t, len(filenames), len(actualFilenames))
		})
	}
}

func TestDumpAndRestore(t *testing.T) {
	changeStorageUsage := func(storageID string, nbytes int64) error { return nil }
	var memStorage []out.LocalStorage
	var diskStorage []out.LocalStorage

	for i := 0; i < 2; i++ {
		mem, _ := NewLocalStorageFS(afero.NewMemMapFs(), changeStorageUsage)
		memStorage = append(memStorage, mem)
	}

	for i := 0; i < 2; i++ {
		disk, _ := NewLocalStorageFS(afero.NewOsFs(), changeStorageUsage)
		diskStorage = append(diskStorage, disk)
	}

	table := []struct {
		storageType string
		storage     []out.LocalStorage
	}{
		{storageType: "memory", storage: memStorage},
		{storageType: "disk", storage: diskStorage},
	}

	for _, v := range table {
		v := v
		t.Run(fmt.Sprintf("dumprestore_%s", v.storageType), func(t *testing.T) {
			filenames := []string{"fileInRoot", "/folder1/file1", "/folder1/file2", "/folder2/file2", "/folder2/folder3/file"}
			for _, filename := range filenames {
				file, err := v.storage[0].Create(filename)
				assert.NoError(t, err)
				file.Close()
			}

			tmpDir := "/tmp/" + uuid.New().String()
			err := os.MkdirAll(tmpDir, 0755)

			if err != nil {
				t.Errorf("failed to create dir. %v", err)
			}

			defer os.RemoveAll(tmpDir)

			err = v.storage[0].DumpToDisk(tmpDir)
			assert.NoError(t, err)

			err = v.storage[1].RestoreFromDisk(tmpDir)
			assert.NoError(t, err)

			actualFilenames, _ := v.storage[1].ListFiles("")
			assert.Equal(t, len(filenames), len(actualFilenames))
		})
	}
}
