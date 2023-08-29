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

//go:generate go run -mod=mod github.com/golang/mock/mockgen -destination=../../../mocks/mock_local_storage_factory.go -package=mocks -source=LocalStorageFactory.go

type LocalStorageFactory interface {
	GetStorageFromID(storageID string) (LocalStorage, error)
	GetLocalStorage(filesize uint64, compressed bool) (LocalStorage, error)
	DestroyStorage(storageID string) error
}
