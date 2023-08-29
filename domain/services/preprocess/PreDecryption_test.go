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
	"context"
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestShouldAskForDecryption(t *testing.T) {
	request := entities.ScanRequest{
		StorageType: "s3",
		Bucket:      "bucket",
		Key:         []string{"pgbackrest/pg-unknown-app/archive/pg-unknown-app/11-1/uuid/uuid.backup"},
	}
	p := NewPreDecryption(logging.NewDiscardLog())
	p.Preprocess(context.Background(), &request)
	assert.Equal(t, 2, len(request.Key))
}

func TestFilterMetadata(t *testing.T) {
	p := NewPreDecryption(logging.NewDiscardLog())
	assert.True(t, p.isMetadata("pgbackrest/pg-app/backup/pg-app/20220319-025518F/backup.info"))
	assert.True(t, p.isMetadata("pgbackrest/pg-app/backup/pg-app/20220319-025518F/backup.manifest"))
	assert.True(t, p.isMetadata("pgbackrest/pg-app/archive/pg-app/archive.info"))
}

func TestFilterNoMetadata(t *testing.T) {
	p := NewPreDecryption(logging.NewDiscardLog())
	assert.False(t, p.isMetadata("pgbackrest/pg-app/backup/pg-app/20220319-025518F/pg_data/backup_label.lz4"))
	assert.False(t, p.isMetadata("pgbackrest/pg-app/archive/pg-app/11-1/0000000100000E90/0000000100000E9000000000.00000028.backup"))
}

func TestShouldNotAskForDecryption(t *testing.T) {
	request := entities.ScanRequest{
		StorageType: "s3",
		Bucket:      "bucket",
		Key:         []string{"backup/test"},
		Size:        100,
	}

	p := NewPreDecryption(logging.NewDiscardLog())
	p.Preprocess(context.Background(), &request)
	assert.Equal(t, 1, len(request.Key))
}
