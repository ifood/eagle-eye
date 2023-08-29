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

package filters

import (
	"context"
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
)

type ProbabilisticFilter struct {
	probabilities map[string]float64
	logger        logging.Logger
}

func NewProbabilisticFilter(probabilities map[string]float64, logger logging.Logger) *ProbabilisticFilter {
	return &ProbabilisticFilter{probabilities: probabilities, logger: logger}
}

func (p *ProbabilisticFilter) Filter(ctx context.Context, request *entities.ScanRequest) entities.JobStatus {
	// For buckets that are listed as probabilistic scan, we must perform a raffle
	if val, ok := p.probabilities[request.Bucket]; ok {
		if !p.shouldScan(val) {
			p.logger.Debugw("File was not selected for scan", "bucket", request.Bucket, "key", request.Key)
			return entities.Abort
		}
	}

	return entities.NextJob
}

func (p *ProbabilisticFilter) shouldScan(scanProbability float64) bool {
	return common.RandFloat64() < scanProbability
}
