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

package cleanup

import (
	"context"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/domain/services/stages"
	"eagle-eye/logging"
)

type StorageCleanup struct {
	localStorageFactory out.LocalStorageFactory
	logger              logging.Logger
}

func NewStorageCleanup(localStorageFactory out.LocalStorageFactory, logger logging.Logger) StorageCleanup {
	return StorageCleanup{localStorageFactory: localStorageFactory, logger: logger}
}

func (s *StorageCleanup) Clean(ctx context.Context, request *stages.Cleanup[entities.ScanRequest]) {
	originalRequest := request.Request
	s.logger.Debugw("delete storage", "storage_id", originalRequest.StorageID)

	err := s.localStorageFactory.DestroyStorage(originalRequest.StorageID)
	if err != nil {
		s.logger.Errorw("failed to delete storage", "error", err, "storage_id", originalRequest.StorageID)
	}
}
