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

package scan

import (
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"fmt"
	"reflect"
	"strings"
	"time"
)
import "context"

type Handler struct {
	asyncInterval time.Duration
	scanService   *Service
	syncScanners  []SyncProcess
	asyncScanners []AsyncProcess
	logger        logging.Logger
}

func NewScanHandler(scanService *Service, syncScanner []SyncProcess, asyncScanner []AsyncProcess, asyncInterval time.Duration, logger logging.Logger) *Handler {
	return &Handler{asyncInterval: asyncInterval, scanService: scanService, syncScanners: syncScanner, asyncScanners: asyncScanner, logger: logger}
}

func (s *Handler) Handle(ctx context.Context, request *entities.ScanRequest, w *entities.OutputWriter[entities.ScanResult]) error {
	result := s.scanService.Scan(ctx, *request)
	w.Write(ctx, &result)

	return fmt.Errorf("enforce cleanup")
}

func (s *Handler) HandleAsync(ctx context.Context, output chan *entities.ScanResult) {
	go func() {
		for range time.After(s.asyncInterval) {
			for _, asyncScanner := range s.asyncScanners {
				s.logger.Debugw("Running job", "job", reflect.ValueOf(asyncScanner).Type())

				results := asyncScanner.GetResults(ctx)
				for _, result := range results {
					result := result
					output <- &result
				}
			}
		}
	}()
}

func (s *Handler) Name() string {
	var jobs []string
	for _, job := range s.syncScanners {
		jobs = append(jobs, reflect.TypeOf(job).Elem().Name())
	}

	return "ScheduleScan Handler with jobs: " + strings.Join(jobs, ", ")
}
