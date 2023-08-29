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

package preprocess

import (
	"context"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/services"
	"eagle-eye/logging"
)

type IndividualScanUpdate struct {
	schedulerService services.Scheduler
	logger           logging.Logger
}

func NewIndividualScanUpdate(schedulerService services.Scheduler, logger logging.Logger) *IndividualScanUpdate {
	i := IndividualScanUpdate{schedulerService: schedulerService, logger: logger}
	return &i
}

func (i *IndividualScanUpdate) Preprocess(ctx context.Context, request *entities.ScanRequest) entities.JobStatus {
	if !i.schedulerService.IsScheduledScan(request.Bucket) {
		return entities.NextJob
	}

	item, err := i.schedulerService.GetSchedule(request.Key[0])
	if err != nil {
		i.logger.Errorw("failed to get scheduled info for scan", "error", err, "request", request)
		return entities.Abort
	}

	if item.Status == entities.Error {
		i.logger.Errorw("failed to get scheduled info for scan", "error", err, "request", request)
		return entities.Abort
	}

	err = i.schedulerService.UpdateSchedule(item.ScanID, entities.Running)
	if err != nil {
		i.logger.Errorw("failed to update schedule info", "error", err, "request", request)
		return entities.Abort
	}

	request.ScanID = item.ScanID
	request.ResultType = entities.Individual
	request.Flags |= entities.DisableVirusTotal

	return entities.NextJob
}
