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
	"eagle-eye/domain/ports/out"
	"eagle-eye/domain/services"
	"reflect"
)

const megabyteBuffer = 1024 * 1024

type Decompress struct {
	localStorageFactory out.LocalStorageFactory
	decompressService   services.DecompressService
}

func NewDecompress(decompressService services.DecompressService, localStorageFactory out.LocalStorageFactory) *Decompress {
	return &Decompress{decompressService: decompressService, localStorageFactory: localStorageFactory}
}

func (d *Decompress) Preprocess(ctx context.Context, request *entities.ScanRequest) entities.JobStatus {
	storage, err := d.localStorageFactory.GetStorageFromID(request.StorageID)
	if err != nil {
		return entities.Abort
	}

	if err := d.decompressService.Extract(storage, make([]byte, megabyteBuffer)); err != nil {
		return entities.Abort
	}

	return entities.NextJob
}

func (d *Decompress) Name() string {
	return reflect.TypeOf(d).Name()
}
