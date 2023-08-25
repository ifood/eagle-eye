//go:build e2e

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

package e2e

import (
	"bytes"
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/go-redis/redis/v9"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"log"
	"os"
	"testing"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"

	"eagle-eye/common"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type E2E struct {
	suite.Suite
	awsStack   *dockertest.Resource
	redisStack *dockertest.Resource

	bucketName string
	queueName  string
	sqsClient  *awssqs.Client
	s3Client   *awss3.Client
}

func TestE2ESuite(t *testing.T) {
	suite.Run(t, new(E2E))
}

func (suite *E2E) SetupSuite() {
	suite.prepareEnvironmentVariables()
	ctx := context.Background()

	pool, err := dockertest.NewPool("")
	suite.Require().NoError(err)

	awsStackConfig := &dockertest.RunOptions{
		Repository:   "localstack/localstack",
		Tag:          "1.0.4",
		Env:          []string{"SERVICES=sqs,s3"},
		ExposedPorts: []string{"4566"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"4566": {{HostIP: "0.0.0.0", HostPort: "4566"}},
		},
	}

	awsStack, err := pool.RunWithOptions(awsStackConfig)
	suite.Require().NoError(err)
	suite.awsStack = awsStack

	redisStackConfig := &dockertest.RunOptions{
		Repository:   "redis",
		Tag:          "6",
		ExposedPorts: []string{"6379"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"6379": {{HostIP: "0.0.0.0", HostPort: "6379"}},
		},
	}

	redisStack, err := pool.RunWithOptions(redisStackConfig)
	suite.Require().NoError(err)
	suite.redisStack = redisStack

	go common.RedirectContainerOutput(ctx, pool, redisStack.Container.ID)
	go common.RedirectContainerOutput(ctx, pool, awsStack.Container.ID)
	time.Sleep(30 * time.Second)

	// Check redis is running
	mockCache := "localhost:6379"
	suite.Require().Eventually(func() bool {
		client := redis.NewClient(&redis.Options{
			Addr:     mockCache,
			Password: "",
			DB:       0,
		})

		_, err := client.Ping(ctx).Result()
		return err == nil
	}, 1*time.Minute, 10*time.Second)

	mockAWSEndpoint := "http://localhost:4566"
	suite.setEnvironmentVariable("AWS_RESOLVER", mockAWSEndpoint)
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			PartitionID:   "aws",
			URL:           mockAWSEndpoint,
			SigningRegion: region,
		}, nil
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithEndpointResolverWithOptions(customResolver),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("key", "secret", "")),
	)

	s3Client := awss3.NewFromConfig(awsCfg, func(o *awss3.Options) { o.UsePathStyle = true })
	sqsClient := awssqs.NewFromConfig(awsCfg)

	// If we can list buckets, then S3 is ready
	suite.Require().Eventually(func() bool {
		_, err = s3Client.ListBuckets(ctx, &awss3.ListBucketsInput{})
		log.Printf("s3: err: %v\n", err)
		return err == nil
	}, 1*time.Minute, 10*time.Second)

	// If we can list queues, then SQS is ready
	suite.Require().Eventually(func() bool {
		_, listErr := sqsClient.ListQueues(ctx, &awssqs.ListQueuesInput{})
		log.Printf("err: %v\n", listErr)
		return listErr == nil
	}, 1*time.Minute, 10*time.Second)

	suite.s3Client = s3Client
	suite.sqsClient = sqsClient

	suite.bucketName = "scanner-samples"
	_, err = suite.s3Client.CreateBucket(ctx, &awss3.CreateBucketInput{Bucket: aws.String(suite.bucketName)})
	suite.Require().NoError(err)

	_, err = suite.s3Client.CreateBucket(ctx, &awss3.CreateBucketInput{Bucket: aws.String("scanner-internal-bucket")})
	suite.Require().NoError(err)

	suite.queueName = "scanner-queue"
	queue, err := suite.sqsClient.CreateQueue(ctx, &awssqs.CreateQueueInput{QueueName: aws.String(suite.queueName)})
	suite.Require().NoError(err)
	suite.Require().Eventually(func() bool {
		_, err = sqsClient.ReceiveMessage(ctx, &awssqs.ReceiveMessageInput{QueueUrl: queue.QueueUrl})
		return err == nil
	}, 1*time.Minute, 10*time.Second)

	suite.setEnvironmentVariable("AWS_QUEUE", *queue.QueueUrl)

	suite.EnableS3ObjectCreateNotifySQS(ctx, "scanner-samples", "arn:aws:sqs:us-east-1:000000000000:scanner-queue")
	suite.EnableS3ObjectCreateNotifySQS(ctx, "scanner-internal-bucket", "arn:aws:sqs:us-east-1:000000000000:scanner-queue")
}

func (suite *E2E) EnableS3ObjectCreateNotifySQS(ctx context.Context, bucketName, queueArn string) {
	_, err := suite.s3Client.PutBucketNotificationConfiguration(ctx, &awss3.PutBucketNotificationConfigurationInput{
		Bucket: aws.String(bucketName),
		NotificationConfiguration: &types.NotificationConfiguration{
			QueueConfigurations: []types.QueueConfiguration{
				{
					Events:   []types.Event{"s3:ObjectCreated:*"},
					QueueArn: aws.String(queueArn),
				},
			},
		},
	})
	suite.Require().NoError(err)
}

func (suite *E2E) TearDownTest() {
	_, err := suite.sqsClient.PurgeQueue(context.Background(), &awssqs.PurgeQueueInput{QueueUrl: &suite.queueName})
	assert.NoError(suite.T(), err)
}

func (suite *E2E) uploadFilesForTest(ctx context.Context, key string, filepath string) {
	body := common.LoadFile(suite.T(), filepath)
	_, err := suite.s3Client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(suite.bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	suite.Require().NoError(err)
}

func (suite *E2E) TearDownSuite() {
	log.Println("finishing e2e tests")
	suite.teardownContainers()
}

func (suite *E2E) teardownContainers() {
	if suite.awsStack != nil {
		suite.Assert().NoError(suite.awsStack.Close())
	}
	if suite.redisStack != nil {
		suite.Assert().NoError(suite.redisStack.Close())
	}
}

func (suite *E2E) setEnvironmentVariable(key, value string) {
	suite.Require().NoError(os.Setenv(key, value))
}

func (suite *E2E) prepareEnvironmentVariables() {
	common.ChangePathForTesting(suite.T())
	suite.setEnvironmentVariable("CONFIG_DIR", "e2e/")
	suite.setEnvironmentVariable("SCANNER_YARA_RULESDIR", "resources/rules/")
	suite.setEnvironmentVariable("AWS_ACCESS_KEY_ID", "key")
	suite.setEnvironmentVariable("AWS_SECRET_ACCESS_KEY", "secret")
}
