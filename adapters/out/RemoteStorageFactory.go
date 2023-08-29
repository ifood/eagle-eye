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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

type RemoteStorageFactory struct {
	storages map[string]out.RemoteStorage
}

func NewRemoteStorageFactory(awsSession *session.Session, awsConfig *aws.Config) *RemoteStorageFactory {
	s3Storage := NewS3Storage(awsSession, awsConfig)
	factory := &RemoteStorageFactory{
		storages: make(map[string]out.RemoteStorage),
	}
	factory.storages["s3"] = s3Storage

	return factory
}

func (r *RemoteStorageFactory) GetRemoteStorage(storageType string) (out.RemoteStorage, error) {
	switch storageType {
	case "s3":
		return r.storages["s3"], nil
	default:
		return nil, fmt.Errorf("there is no such storage type %s", storageType)
	}
}
