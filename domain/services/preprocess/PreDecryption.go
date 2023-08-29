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
	"fmt"
	"reflect"
	"regexp"
)

const expectedNumberMatches = 2

type PreDecryption struct {
	logger logging.Logger
}

func NewPreDecryption(logger logging.Logger) *PreDecryption {
	return &PreDecryption{logger: logger}
}

func (p *PreDecryption) Preprocess(ctx context.Context, request *entities.ScanRequest) entities.JobStatus {
	if p.isMetadata(request.Key[0]) {
		return entities.NextJob
	}

	additionalFiles := p.getAdditionalFiles(request.Key[0])
	request.Key = append(request.Key, additionalFiles...)

	return entities.NextJob
}

func (p *PreDecryption) isMetadata(key string) bool {
	return p.hasPattern(`pgbackrest/.*/(archive.info|backup.info|backup.manifest)`, key)
}

func (p *PreDecryption) isArchive(key string) bool {
	return p.hasPattern(`pgbackrest/(.*?)/archive/.*?/.*`, key)
}

func (p *PreDecryption) isBackup(key string) bool {
	return p.hasPattern(`pgbackrest/(.*?)/backup/.*/pg_data/.*`, key)
}

func (p *PreDecryption) hasPattern(pattern, key string) bool {
	hasPattern := true
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(key)

	if len(match) != expectedNumberMatches {
		p.logger.Debugw("failed to identify pattern in path", "pattern", pattern, "key", key)
		hasPattern = false
	}

	return hasPattern
}

func (p *PreDecryption) getAdditionalFiles(key string) []string {
	if p.isArchive(key) {
		return []string{p.getArchiveMetadata(key)}
	}

	if p.isBackup(key) {
		return p.getBackupMetadata(key)
	}

	return []string{}
}

func (p *PreDecryption) getArchiveMetadata(key string) string {
	re := regexp.MustCompile(`pgbackrest/(.*?)/.*`)
	match := re.FindStringSubmatch(key)

	return fmt.Sprintf("pgbackrest/%s/archive/%s/archive.info", match[1], match[1])
}

func (p *PreDecryption) getBackupMetadata(key string) []string {
	re := regexp.MustCompile(`pgbackrest/(.*?)/backup/.*?/(.*?)/pg_data/.*`)
	match := re.FindStringSubmatch(key)
	backupInfo := fmt.Sprintf("pgbackrest/%s/backup/%s/%s/backup.info", match[1], match[1], match[2])
	backupManifest := fmt.Sprintf("pgbackrest/%s/backup/%s/%s/backup.manifest", match[1], match[1], match[2])

	return []string{backupManifest, backupInfo}
}

func (p *PreDecryption) Name() string {
	return reflect.TypeOf(p).Name()
}
