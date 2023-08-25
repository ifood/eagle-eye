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

package notification

import (
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/logging"
	"fmt"
	"sync"
)

type EmergencyService struct {
	mu               sync.Mutex
	matchesPerBucket map[string]int
	viewers          []out.Viewer
	logger           logging.Logger
}

func NewEmergencyService(viewers []out.Viewer, logger logging.Logger) *EmergencyService {
	return &EmergencyService{matchesPerBucket: make(map[string]int), viewers: viewers, logger: logger}
}

func (e *EmergencyService) Update(result entities.ScanResult) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Rest API calls should not trigger emergency notifications.
	if result.ResultType == entities.Individual || result.Matches == 0 {
		return
	}

	e.matchesPerBucket[result.Bucket] += result.Matches
}

func (e *EmergencyService) UpdateGlobal() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.matchesPerBucket) == 0 {
		return
	}

	message := "Malicious artifacts detected in the following buckets, please check the logs for more information:\n"
	for bucket, matches := range e.matchesPerBucket {
		message += fmt.Sprintf("%s -> %d\n", bucket, matches)
	}

	for _, viewer := range e.viewers {
		viewer.SendMessage(message)
	}

	// Require at least a single success to clean the results
	e.matchesPerBucket = make(map[string]int)
}
