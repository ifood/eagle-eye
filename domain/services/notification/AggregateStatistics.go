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
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/logging"
	"reflect"
	"sync"
)

type AggregateStatistics struct {
	stats  map[string]entities.ScanResult
	mu     sync.Mutex
	repo   out.AggregateScanRepository
	logger logging.Logger
}

func NewAggregateStatistics(repo out.AggregateScanRepository, logger logging.Logger) *AggregateStatistics {
	return &AggregateStatistics{repo: repo, stats: make(map[string]entities.ScanResult), logger: logger}
}

func (a *AggregateStatistics) Update(result entities.ScanResult) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if result.ResultType != entities.Aggregate {
		return
	}

	if val, ok := a.stats[result.Bucket]; ok {
		a.stats[result.Bucket] = entities.MergeScanResults(val, result)
	} else {
		a.stats[result.Bucket] = result
	}
}

func (a *AggregateStatistics) UpdateGlobal() {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, value := range a.stats {
		func() {
			previousResult, err := a.repo.GetByBucketAndDate(value.Bucket, value.LastUpdate.Day(), int(value.LastUpdate.Month()))
			if err != nil {
				a.logger.Errorw("Could not obtain previous result", "error", err)
				return
			}

			updatedResult := entities.ScanResult{Bucket: value.Bucket, Entropy: common.CreateEmptyEntropyBuckets()}
			updatedResult = entities.MergeScanResults(updatedResult, previousResult)
			updatedResult = entities.MergeScanResults(updatedResult, value)

			err = a.repo.Save(updatedResult)
			if err != nil {
				a.logger.Errorw("failed to save updated bucket result", "error", err, "bucket", value.Bucket, "result", value)
				return
			}

			delete(a.stats, value.Bucket)
		}()
	}
}

func (a *AggregateStatistics) Name() string {
	return reflect.TypeOf(a).Name()
}
