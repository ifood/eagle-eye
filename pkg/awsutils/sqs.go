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
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	maxMessagesToFetch = 10
	poolWaitTime       = 20
)

type SQS struct {
	svc *sqs.SQS
}

func (s *SQS) Init(awsSession *session.Session, awsConfig *aws.Config) {
	s.svc = sqs.New(awsSession, awsConfig)
}

func (s *SQS) ReceiveMessageFromSQS(queueURL string) ([]*sqs.Message, error) {
	result, err := s.svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
		},
		QueueUrl:            &queueURL,
		MaxNumberOfMessages: aws.Int64(maxMessagesToFetch),
		WaitTimeSeconds:     aws.Int64(poolWaitTime),
	})

	if err != nil {
		return nil, err
	}

	if len(result.Messages) == 0 {
		return nil, nil
	}

	return result.Messages, nil
}

func (s *SQS) DeleteMessageFromSQS(queueURL string, message *sqs.Message) error {
	_, err := s.svc.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      &queueURL,
		ReceiptHandle: message.ReceiptHandle,
	})

	if err != nil {
		return err
	}

	return nil
}
