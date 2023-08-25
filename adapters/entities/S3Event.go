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

package entities

type S3Event struct {
	AwsRegion string `json:"awsRegion"`
	EventName string `json:"eventName"`
	S3        S3     `json:"s3"`
}

type S3 struct {
	Bucket Bucket `json:"bucket"`
	Object Object
}

type Bucket struct {
	Name string `json:"name"`
}

type Object struct {
	Key  string `json:"key"`
	Size uint64 `json:"size"`
}
