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
	"eagle-eye/crypto"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/logging"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
)

const (
	bufferSize              = 1024 * 1024
	expectedNumberOfMatches = 2
	saltSize                = 16
)

type PosDecryption struct {
	decryptionKEK       string
	localStorageFactory out.LocalStorageFactory
	logger              logging.Logger
}

func NewPostDecryption(localStorageFactory out.LocalStorageFactory, decryptionKEK string, logger logging.Logger) *PosDecryption {
	return &PosDecryption{decryptionKEK: decryptionKEK, localStorageFactory: localStorageFactory, logger: logger}
}

func (p *PosDecryption) Preprocess(ctx context.Context, request *entities.ScanRequest) entities.JobStatus {
	storage, err := p.localStorageFactory.GetStorageFromID(request.StorageID)
	if err != nil {
		p.logger.Errorw("Failed to get local storage", "error", err, "storageId", request.StorageID, "bucket", request.Bucket, "key", request.Key)
		return entities.NextJob
	}

	outputKey := p.decrypt(request.Key, storage)
	request.Key = []string{outputKey}

	return entities.NextJob
}

func (p *PosDecryption) decryptArchive(keys []string, storage out.LocalStorage) string {
	passphrase, err := p.extractPasswordFromFile(p.decryptionKEK, keys[1], storage)
	if err != nil {
		p.logger.Errorw("Cant extract passphrase", "error", err, "key", keys[1])
	}

	filename, err := p.decryptEndfile(passphrase, keys[0], storage)
	if err != nil {
		p.logger.Errorw("Cant decrypt file", "error", err, "key", keys[0])
	}

	return filename
}

func (p *PosDecryption) decryptBackup(keys []string, storage out.LocalStorage) string {
	passphrase, err := p.extractPasswordFromFile(p.decryptionKEK, keys[2], storage)
	if err != nil {
		p.logger.Errorw("Cant extract passphrase", "error", err, "key", keys[2])
	}

	passphrase, err = p.extractPasswordFromFile(passphrase, keys[1], storage)
	if err != nil {
		p.logger.Errorw("Cant extract passphrase", "error", err, "key", keys[1])
	}

	filename, err := p.decryptEndfile(passphrase, keys[0], storage)
	if err != nil {
		p.logger.Errorw("Cant decrypt file", "error", err, "key", keys[0])
	}

	return filename
}

func (p *PosDecryption) loadFile(filename string, storage out.LocalStorage) ([]byte, error) {
	encryptedBackup, err := storage.Open(filename)
	if err != nil {
		return nil, err
	}
	defer encryptedBackup.Close()

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, encryptedBackup)

	return buf.Bytes(), err
}

func (p *PosDecryption) decrypt(keys []string, storage out.LocalStorage) string {
	if p.isArchive(keys[0]) {
		return p.decryptArchive(keys, storage)
	}

	if p.isBackup(keys[0]) {
		return p.decryptBackup(keys, storage)
	}

	return keys[0]
}

func (p *PosDecryption) isArchive(key string) bool {
	return p.hasPattern(`pgbackrest/(.*?)/archive/.*?/.*`, key)
}

func (p *PosDecryption) isBackup(key string) bool {
	return p.hasPattern(`pgbackrest/(.*?)/backup/.*/pg_data/.*`, key)
}

func (p *PosDecryption) hasPattern(pattern, key string) bool {
	hasPattern := true
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(key)

	if len(match) != expectedNumberOfMatches {
		p.logger.Debugw("failed to identify pattern in path", "pattern", pattern, "path", key)
		hasPattern = false
	}

	return hasPattern
}

func (p *PosDecryption) decryptEndfile(passphrase, filename string, storage out.LocalStorage) (string, error) {
	decryptedFilename := fmt.Sprintf("decrypted-%s", filename)
	cleartextFile, err := storage.Create(decryptedFilename)

	if err != nil {
		return filename, fmt.Errorf("failed to create decrypted file %s. %w", filename, err)
	}

	defer func() {
		err = cleartextFile.Close()
		p.logger.Errorw("failed to close file.", "error", err)
	}()

	encryptedFile, err := storage.Open(filename)
	if err != nil {
		return filename, fmt.Errorf("failed to open encrypted file %s. %w", filename, err)
	}
	defer encryptedFile.Close()

	err = p.decryptLargeFile(passphrase, encryptedFile, cleartextFile)
	if err != nil {
		return filename, err
	}

	return decryptedFilename, nil
}

//nolint:cyclop
func (p *PosDecryption) decryptLargeFile(passphrase string, reader io.Reader, writer io.Writer) error {
	salt := make([]byte, saltSize)
	if _, err := reader.Read(salt); err != nil {
		return fmt.Errorf("failed to read salt from file %w", err)
	}

	decryptionEngine, err := crypto.NewDecryptionEngine(passphrase, salt[8:])
	if err != nil {
		return fmt.Errorf("failed to create key and iv for file. %w", err)
	}

	buf := bytes.NewBuffer(make([]byte, bufferSize))
	stop := false

	for !stop {
		buf.Reset()
		n, err := io.CopyN(buf, reader, bufferSize)

		if errors.Is(err, io.EOF) {
			stop = true
		}

		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to extract file. %w", err)
		}

		cleartextBlock, err := decryptionEngine.DecryptBlock(buf.Bytes()[:n])
		if err != nil {
			return fmt.Errorf("failed while decrypting. %w", err)
		}

		_, err = writer.Write(cleartextBlock)
		if err != nil {
			return fmt.Errorf("failed while writing. %w", err)
		}
	}

	cleartextBlock, err := decryptionEngine.DecryptEnd()
	if err != nil {
		return fmt.Errorf("failed to end decryption procedure. %w", err)
	}

	_, err = writer.Write(cleartextBlock)
	if err != nil {
		return fmt.Errorf("failed to write decrypted info to file. %w", err)
	}

	return nil
}

func (p *PosDecryption) extractPasswordFromFile(passphrase, filename string, storage out.LocalStorage) (string, error) {
	encryptedBackupInfo, err := p.loadFile(filename, storage)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s. %w", filename, err)
	}

	decryptedBackupInfo, err := crypto.Decrypt(passphrase, encryptedBackupInfo)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt file %s. %w", filename, err)
	}

	passphrase, err = p.extractPassword(string(decryptedBackupInfo))
	if err != nil {
		return "", fmt.Errorf("failed to extract passphrase from file %s. %w", filename, err)
	}

	return passphrase, nil
}

func (p *PosDecryption) extractPassword(data string) (string, error) {
	var re = regexp.MustCompile(`.*cipher-pass.*="(.*)"`)
	matches := re.FindAllStringSubmatch(data, -1)

	if len(matches) > 0 && len(matches[0]) == expectedNumberOfMatches {
		return matches[0][1], nil
	}

	return "", fmt.Errorf("failed to extract backup's password")
}

func (p *PosDecryption) Name() string {
	return reflect.TypeOf(p).Name()
}
