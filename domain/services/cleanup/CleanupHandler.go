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
	"reflect"
	"strings"
)

type Job interface {
	Clean(cxt context.Context, request *stages.Cleanup[entities.ScanRequest])
}

type Handler struct {
	jobs   []Job
	logger logging.Logger
}

func NewCleanupHandler(cleanupJobs []Job, logger logging.Logger) *Handler {
	return &Handler{
		logger: logger,
		jobs:   cleanupJobs,
	}
}

func (c *Handler) Handle(ctx context.Context, request *stages.Cleanup[entities.ScanRequest], w *entities.OutputWriter[entities.CleanRequest]) error {
	for _, job := range c.jobs {
		c.logger.Debugw("Running job", "job", reflect.ValueOf(job).Type())
		job.Clean(ctx, request)
	}

	return nil
}

func (c *Handler) Name() string {
	var jobs []string
	for _, job := range c.jobs {
		jobs = append(jobs, reflect.TypeOf(job).Elem().Name())
	}

	return "Cleanup Handler with jobs: " + strings.Join(jobs, ", ")
}
