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
	"eagle-eye/logging"
	"reflect"
	"strings"
	"time"
)
import "context"

type Handler struct {
	jobs   []Job
	logger logging.Logger
}

type Job interface {
	Update(result entities.ScanResult)
	UpdateGlobal()
}

func NewNotificationHandler(notifiers []Job, logger logging.Logger) *Handler {
	return &Handler{jobs: notifiers, logger: logger}
}

func (n *Handler) Handle(ctx context.Context, request *entities.ScanResult, _ *entities.OutputWriter[entities.Empty]) error {
	for _, notifier := range n.jobs {
		n.logger.Debugw("Running job", "job", reflect.ValueOf(notifier).Type())
		notifier.Update(*request)
	}

	return nil
}

func (n *Handler) HandleAsync(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				n.logger.Infow("Notifying external systems before termination")

				for _, notifier := range n.jobs {
					notifier.UpdateGlobal()
				}

				n.logger.Infow("Notifying external systems before termination completed")

				return
			case <-ticker.C:
				for _, notifier := range n.jobs {
					notifier.UpdateGlobal()
				}
			}
		}
	}()
}

func (n *Handler) Name() string {
	var jobs []string
	for _, job := range n.jobs {
		jobs = append(jobs, reflect.TypeOf(job).Elem().Name())
	}

	return "Notification Handler with jobs: " + strings.Join(jobs, ", ")
}
