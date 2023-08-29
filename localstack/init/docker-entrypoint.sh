#!/bin/bash

#
#    Copyright 2023 iFood
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
#

# Create notification queue
aws --endpoint-url=http://127.0.0.1:4566 sqs create-queue --queue-name scanner-queue

# Get queue arn
aws --endpoint-url=http://127.0.0.1:4566 sqs get-queue-attributes --queue-url http://localhost:4576/queue/scanner-queue --attribute-names All --output text --query 'Attributes.QueueArn'

# Create bucket for samples
aws --endpoint-url=http://127.0.0.1:4566 s3api create-bucket --bucket samples-scanner-bucket

# Create internal scanner bucket
aws --endpoint-url=http://127.0.0.1:4566 s3api create-bucket --bucket scanner-internal-bucket

# Configure notification on samples-scanner-bucket
aws --endpoint-url=http://127.0.0.1:4566 s3api put-bucket-notification-configuration --bucket samples-scanner-bucket --notification-configuration file:///docker-entrypoint-initaws.d/notification.json

# Configure notification on internal scanner bucket
aws --endpoint-url=http://127.0.0.1:4566 s3api put-bucket-notification-configuration --bucket scanner-internal-bucket --notification-configuration file:///docker-entrypoint-initaws.d/notification.json

# Check configuration was set
aws --endpoint-url=http://127.0.0.1:4566 s3api get-bucket-notification-configuration --bucket samples-scanner-bucket