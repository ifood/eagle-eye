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

package cleanup

import (
	"context"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/services/stages"
	"eagle-eye/logging"
	"eagle-eye/pkg/awsutils"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type QueueCleanup struct {
	queue      string
	sqsService awsutils.SQS
	logger     logging.Logger
}

func NewQueueCleanup(queue string, sqsService awsutils.SQS, logger logging.Logger) QueueCleanup {
	return QueueCleanup{logger: logger, queue: queue, sqsService: sqsService}
}

func (q *QueueCleanup) Clean(ctx context.Context, request *stages.Cleanup[entities.ScanRequest]) {
	originalRequest := request.Request
	if originalRequest.MessageID != "" {
		q.logger.Debugw("Deleting message", "message_id", originalRequest.MessageID)
		message := sqs.Message{ReceiptHandle: &originalRequest.MessageID}
		err := q.sqsService.DeleteMessageFromSQS(q.queue, &message)

		if err != nil {
			q.logger.Errorw("failed to delete message from sqs service", "error", err)
		}
	}
}
