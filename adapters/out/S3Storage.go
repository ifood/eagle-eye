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
	"eagle-eye/pkg/awsutils"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"io"
)

type S3Storage struct {
	svc awsutils.S3
}

func NewS3Storage(awsSession *session.Session, awsConfig *aws.Config) *S3Storage {
	svc := awsutils.S3{}
	svc.Init(awsSession, awsConfig)

	return &S3Storage{svc: svc}
}

func (s *S3Storage) Get(bucket, name string, writer io.WriterAt) error {
	return s.svc.DownloadFromS3Bucket(writer, bucket, name, "")
}

func (s *S3Storage) GetHeader(bucket, name string, size uint64, writer io.WriterAt) error {
	return s.svc.DownloadFromS3Bucket(writer, bucket, name, fmt.Sprintf("bytes=0-%d", size))
}

func (s *S3Storage) Put(bucket, name string, reader io.Reader) error {
	return s.svc.UploadToS3Bucket(reader, bucket, name)
}
