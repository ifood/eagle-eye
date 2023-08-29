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

import (
	"encoding/json"
	"time"
)

type Status string

const (
	Waiting   Status = "waiting"
	Running   Status = "running"
	Completed Status = "completed"
	Error     Status = "error"
)

type ScheduleItem struct {
	ScanID     string
	Bucket     string
	Key        string
	Filename   string
	CreateTime time.Time
}

type ScheduleItemWithState struct {
	ScanID     string
	Bucket     string
	Filename   string
	Key        string
	CreateTime time.Time
	LastUpdate time.Time
	Status     Status
}

func (s ScheduleItemWithState) MarshalBinary() (data []byte, err error) {
	bytes, err := json.Marshal(s)
	return bytes, err
}
