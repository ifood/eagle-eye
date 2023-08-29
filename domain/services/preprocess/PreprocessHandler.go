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
	"eagle-eye/logging"
	"fmt"
	"reflect"
	"strings"
)

type Handler struct {
	jobs   []Job
	logger logging.Logger
}

type Job interface {
	Preprocess(ctx context.Context, request *entities.ScanRequest) entities.JobStatus
}

func NewPreprocessHandler(jobs []Job, logger logging.Logger) *Handler {
	return &Handler{jobs: jobs, logger: logger}
}

func (p *Handler) Handle(ctx context.Context, request *entities.ScanRequest, w *entities.OutputWriter[entities.ScanRequest]) error {
	shouldScan := entities.NextJob

	for _, preprocessor := range p.jobs {
		p.logger.Debugw("Running job", "job", reflect.ValueOf(preprocessor).Type())
		shouldScan = preprocessor.Preprocess(ctx, request)

		if shouldScan == entities.NextStage {
			w.Write(ctx, request)
			break
		}

		if shouldScan == entities.Abort {
			return fmt.Errorf("preprocess error")
		}
	}

	if shouldScan == entities.NextJob {
		w.Write(ctx, request)
	}

	return nil
}

func (p *Handler) Name() string {
	var jobs []string
	for _, job := range p.jobs {
		jobs = append(jobs, reflect.TypeOf(job).Elem().Name())
	}

	return "Preprocess Handler with jobs: " + strings.Join(jobs, ", ")
}
