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

import "io"

type RemoteStorage interface {
	RemoteStorageReader
	RemoteStorageWriter
}

// Interface to be implemented by AWS S3, HTTPS file location etc
type RemoteStorageReader interface {
	Get(bucket, name string, writer io.WriterAt) error
	GetHeader(bucket, name string, size uint64, writer io.WriterAt) error
}

type RemoteStorageWriter interface {
	Put(bucket, name string, reader io.Reader) error
}
