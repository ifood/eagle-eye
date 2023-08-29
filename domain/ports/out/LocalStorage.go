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

import "github.com/spf13/afero"

// Interface to be implemented by RAM and Disk storages
// We'll need a factory because we want to use both types
// Implementation should have a limit size
type LocalStorage interface {
	GetID() string
	Destroy() error
	RestoreFromDisk(src string) error
	DumpToDisk(dst string) error
	ListFiles(path string) ([]string, error)
	Exists(name string) (bool, error)
	IsRegular(name string) (bool, error)
	Size(name string) (int64, error)
	afero.Fs
}
