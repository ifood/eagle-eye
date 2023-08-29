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

package preprocess

import (
	"bytes"
	"context"
	"eagle-eye/adapters/out"
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
)

func TestArchiveDecrypt(t *testing.T) {
	localStorageFactory := out.NewLocalStorageFactory(1024 * 1024 * 1024)
	localStorage, _ := localStorageFactory.GetLocalStorage(1024, false)

	d := NewPostDecryption(localStorageFactory, "passkek", logging.NewDiscardLog())

	// Archive info (fake archive info)
	info, _ := localStorage.Create("archive.info")
	common.LoadFileToStorage(t, "archive.info", info)
	info.Close()

	// Encrypted data
	data, _ := localStorage.Create("pgbackrest/pg-app/archive/pg-app/encdata")
	common.LoadFileToStorage(t, "encdata", data)
	data.Close()

	request := entities.ScanRequest{
		Key:       []string{"pgbackrest/pg-app/archive/pg-app/encdata", "archive.info"},
		StorageID: localStorage.GetID(),
	}
	d.Preprocess(context.Background(), &request)

	b := bytes.Buffer{}
	data, _ = localStorage.Open(request.Key[0])
	_, err := io.Copy(&b, data)

	assert.NoError(t, err)
	assert.Equal(t, []byte("cleartext content"), b.Bytes())
}

func TestBackupDecrypt(t *testing.T) {
	localStorageFactory := out.NewLocalStorageFactory(1024 * 1024 * 1024)
	localStorage, _ := localStorageFactory.GetLocalStorage(5*1024*1024, false)

	d := NewPostDecryption(localStorageFactory, "passkek", logging.NewDiscardLog())

	backupInfo, _ := localStorage.Create("backup.info")
	common.LoadFileToStorage(t, "backup.info", backupInfo)
	backupInfo.Close()

	backupManifest, _ := localStorage.Create("backup.manifest")
	common.LoadFileToStorage(t, "backup.manifest", backupManifest)
	backupManifest.Close()

	encData, _ := localStorage.Create("pgbackrest/pg-app/backup/pg-app/20220319-025518F/pg_data/pg_hba.conf.lz4")
	common.LoadFileToStorage(t, "pg_hba.conf.lz4", encData)
	encData.Close()

	request := entities.ScanRequest{
		Key:       []string{"pgbackrest/pg-app/backup/pg-app/20220319-025518F/pg_data/pg_hba.conf.lz4", "backup.manifest", "backup.info"},
		StorageID: localStorage.GetID(),
	}

	d.Preprocess(context.Background(), &request)
	b := bytes.Buffer{}
	output, _ := localStorage.Open(request.Key[0])
	_, err := io.Copy(&b, output)

	assert.NoError(t, err)
	assert.Equal(t, []byte("cleartext content"), b.Bytes())
}
