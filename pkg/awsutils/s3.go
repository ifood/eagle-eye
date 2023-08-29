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

package awsutils

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"io"
	"net/url"
)

const (
	downloadConcurrency = 1 // Disabling concurrency because of memory storage. TODO: Enable it
	downloadPartSize    = 64 * 1024 * 1024
	uploadConcurrency   = 4
)

type S3 struct {
	svc        *s3.S3
	downloader *s3manager.Downloader
	uploader   *s3manager.Uploader
}

func (s *S3) Init(awsSession *session.Session, awsConfig *aws.Config) {
	s.svc = s3.New(awsSession, awsConfig)

	s.downloader = s3manager.NewDownloaderWithClient(s.svc, func(d *s3manager.Downloader) {
		d.Concurrency = downloadConcurrency
	})

	s.uploader = s3manager.NewUploaderWithClient(s.svc, func(u *s3manager.Uploader) {
		u.PartSize = downloadPartSize
		u.Concurrency = uploadConcurrency
	})
}

func (s *S3) ListFilesFromS3Bucket(bucket, prefix string, token *string) (*s3.ListObjectsV2Output, error) {
	items, err := s.svc.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:            aws.String(bucket),
		Prefix:            aws.String(prefix),
		ContinuationToken: token,
	})

	return items, err
}

// Downloads a file from S3 using some paralellism.
// Refs https://github.com/awsdocs/aws-doc-sdk-examples/blob/main/go/example_code/s3/s3_download_object.go
// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/s3manager/#Downloader
func (s *S3) DownloadFromS3Bucket(file io.WriterAt, bucket, item, rangeHeader string) error {
	// Some items have URL encoded parts that were causing download issues.
	item, err := url.QueryUnescape(item)
	if err != nil {
		return err
	}

	object := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(item),
		Range:  aws.String(rangeHeader),
	}

	_, err = s.downloader.Download(file, object)

	return err
}

// Writes file to AWS using some parallelism.
// https://www.matscloud.com/docs/cloud-sdk/go-and-s3/
func (s *S3) UploadToS3Bucket(data io.Reader, bucket, key string) error {
	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   data,
	})

	return err
}

// Get tag from AWS object
func (s *S3) GetTagsFromObject(bucket, key string) (*s3.GetObjectTaggingOutput, error) {
	tag, err := s.svc.GetObjectTagging(&s3.GetObjectTaggingInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	return tag, err
}
