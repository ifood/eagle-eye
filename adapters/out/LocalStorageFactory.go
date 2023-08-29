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
	"eagle-eye/domain/ports/out"
	"fmt"
	"github.com/spf13/afero"
	"sync"
)

const (
	maxSizeForMemory     = 1 * 1024 * 1024
	EnforceDiskSize      = maxSizeForMemory + 1
	noStorageConsumption = 0
)

type LocalStorageFactory struct {
	storage             map[string]out.LocalStorage
	storageUsage        map[string]int64
	maxStorageUsage     int64
	currentStorageUsage int64
	lock                sync.RWMutex
}

func NewLocalStorageFactory(maxStorageUsage int64) *LocalStorageFactory {
	return &LocalStorageFactory{maxStorageUsage: maxStorageUsage, storage: make(map[string]out.LocalStorage), storageUsage: make(map[string]int64)}
}

func (l *LocalStorageFactory) GetLocalStorage(filesize uint64, compressed bool) (out.LocalStorage, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	var storageID string
	var storage out.LocalStorage
	var err error

	if filesize <= maxSizeForMemory && !compressed {
		storage, err = NewLocalStorageFS(afero.NewMemMapFs(), l.changeFSUsage)
	} else {
		storage, err = NewLocalStorageFS(afero.NewOsFs(), l.changeFSUsage)
	}

	if err != nil {
		return nil, err
	}

	storageID = storage.GetID()
	l.storage[storageID] = storage
	l.storageUsage[storageID] = noStorageConsumption

	return l.storage[storageID], nil
}

func (l *LocalStorageFactory) changeFSUsage(storageID string, delta int64) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if _, ok := l.storageUsage[storageID]; !ok {
		return fmt.Errorf("storage not found")
	}

	if l.currentStorageUsage+delta > l.maxStorageUsage {
		return fmt.Errorf("max memory consumed")
	}

	l.currentStorageUsage += delta
	l.storageUsage[storageID] += delta

	return nil
}

func (l *LocalStorageFactory) DestroyStorage(storageID string) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	if _, ok := l.storage[storageID]; !ok {
		return fmt.Errorf("storage not found")
	}

	storage := l.storage[storageID]
	delete(l.storage, storageID)
	err := storage.Destroy()

	if err != nil {
		return err
	}

	if _, ok := l.storageUsage[storageID]; !ok {
		return fmt.Errorf("storage usage not found")
	}

	l.currentStorageUsage -= l.storageUsage[storageID]
	delete(l.storageUsage, storageID)

	return nil
}

func (l *LocalStorageFactory) GetStorageFromID(storageID string) (out.LocalStorage, error) {
	l.lock.RLock()
	defer l.lock.RUnlock()

	if storage, ok := l.storage[storageID]; ok {
		return storage, nil
	}

	return nil, fmt.Errorf("storage not found")
}
