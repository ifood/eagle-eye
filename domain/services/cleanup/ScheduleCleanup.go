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
	"eagle-eye/domain/services"
	"eagle-eye/domain/services/stages"
	"eagle-eye/logging"
)

type ScheduleCleanup struct {
	schedulerService services.Scheduler
	logger           logging.Logger
}

func NewScheduleCleanup(schedulerService services.Scheduler, logger logging.Logger) *ScheduleCleanup {
	s := ScheduleCleanup{schedulerService: schedulerService, logger: logger}
	return &s
}

func (s *ScheduleCleanup) Clean(ctx context.Context, request *stages.Cleanup[entities.ScanRequest]) {
	originalRequest := request.Request
	if !s.schedulerService.IsScheduledScan(originalRequest.Bucket) {
		return
	}

	err := s.schedulerService.UpdateSchedule(originalRequest.ScanID, entities.Completed)
	if err != nil {
		s.logger.Errorw("failed to update schedule scan status", "error", err)
	}
}
