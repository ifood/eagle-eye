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

package in

import (
	"context"
	adapterentities "eagle-eye/adapters/entities"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/fileutils"
	"eagle-eye/logging"
	"eagle-eye/pkg/awsutils"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
	"github.com/uber-go/tally/v4"
	"strings"
)

const (
	consumeCount     = "consume_count"
	singleMessageInc = 1
)

type QueueController struct {
	outputChannel       chan *entities.ScanRequest
	localStorageFactory out.LocalStorageFactory

	sqsService awsutils.SQS
	queue      string

	logger       logging.Logger
	metricsScope tally.Scope
}

func NewQueueController(queue string, localStorageFactory out.LocalStorageFactory, outputChannel chan *entities.ScanRequest, sqsService awsutils.SQS, metricsScope tally.Scope, logger logging.Logger) QueueController {
	return QueueController{queue: queue, localStorageFactory: localStorageFactory, outputChannel: outputChannel, sqsService: sqsService, logger: logger, metricsScope: metricsScope}
}

func (q *QueueController) AsyncScan(ctx context.Context) {
	if q.queue == "" {
		q.logger.Infow("Won't attempt to read SQS queue, because none was configured")
		return
	}

	q.logger.Infow("Start of async queue processing")

	for {
		select {
		case <-ctx.Done():
			q.logger.Infow("End of async queue processing")
			return

		default:
			messages, err := q.sqsService.ReceiveMessageFromSQS(q.queue)
			if err != nil {
				// TODO: should wait a little
				q.logger.Errorw("failed to obtain scan request", "error", err)
				continue
			}

			for _, m := range messages {
				events, err := q.extractEvents(m)
				if err != nil {
					q.logger.Errorw("failed to extract events", "error", err)
					continue
				}

				for _, event := range events {
					q.submitSingleFileForAnalysis(event, *m.ReceiptHandle)
				}
			}
		}
	}
}

func (q *QueueController) extractEvents(m *sqs.Message) ([]adapterentities.S3Event, error) {
	var notification adapterentities.SQSNotification

	// extract sqs message Body
	err := json.Unmarshal([]byte(*m.Body), &notification)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message. %w", err)
	}

	// extract events
	var events adapterentities.Events
	if err = json.Unmarshal([]byte(notification.Message), &events); err != nil {
		// Attempt second type of urmarshalling, only needed for localstack
		err = json.Unmarshal([]byte(*m.Body), &events)
	}

	// Message could not be decoded, should be removed from the queue.
	// TODO: Just ignore it, so it'll go to the DQL
	if err != nil {
		q.logger.Errorw("failed to unmarshal message.", "error", err, "message field", notification.Message, "message", m)

		err = q.sqsService.DeleteMessageFromSQS(q.queue, m)
		if err != nil {
			q.logger.Errorw("deleting invalid message from sqs service failed", "error", err, "message", m)
		}

		return nil, err
	}

	return events.Record, nil
}

func (q *QueueController) submitSingleFileForAnalysis(event adapterentities.S3Event, messageID string) {
	if !strings.HasPrefix(event.EventName, "ObjectCreated:") {
		return
	}

	q.logger.Debugw("Received new request", "region", event.AwsRegion, "bucket", event.S3.Bucket.Name, "key", event.S3.Object.Key, "size", event.S3.Object.Size)

	uniqueUUID, err := uuid.NewRandom()
	if err != nil {
		q.logger.Errorw("Failed to generate scanID for file", "error", err, "region", event.AwsRegion, "bucket", event.S3.Bucket.Name, "key", event.S3.Object.Key, "size", event.S3.Object.Size)
		return
	}

	storage, err := q.localStorageFactory.GetLocalStorage(event.S3.Object.Size, fileutils.IsCompressed(event.S3.Object.Key))
	if err != nil {
		q.logger.Errorw("Failed to create storage for request", "error", err, "region", event.AwsRegion, "bucket", event.S3.Bucket.Name, "key", event.S3.Object.Key, "size", event.S3.Object.Size)
		return
	}

	q.outputChannel <- &entities.ScanRequest{
		ResultType:  entities.Aggregate,
		ScanID:      uniqueUUID.String(),
		StorageType: "s3",
		StorageID:   storage.GetID(),
		Key:         []string{event.S3.Object.Key},
		Bucket:      event.S3.Bucket.Name,
		Size:        event.S3.Object.Size,
		MessageID:   messageID,
	}

	q.metricsScope.Counter(consumeCount).Inc(singleMessageInc)
}
