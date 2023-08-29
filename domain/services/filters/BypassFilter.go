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
	"eagle-eye/domain/entities"
	"eagle-eye/logging"
	"strings"
)

type BypassFilter struct {
	allowlist map[string][]string
	sizeLimit uint64
	logger    logging.Logger
}

func NewBypassfilter(allowlist map[string][]string, sizeLimit uint64, logger logging.Logger) *BypassFilter {
	for bucket, prefixes := range allowlist {
		for _, prefix := range prefixes {
			if prefix[len(prefix)-1] != '/' {
				logger.Infow("Allowlist has permission ending with /, which may cause undesired behavior", "bucket", bucket, "prefix", prefix)
			}
		}
	}

	return &BypassFilter{allowlist: allowlist, sizeLimit: sizeLimit, logger: logger}
}

func (b *BypassFilter) Filter(ctx context.Context, request *entities.ScanRequest) entities.JobStatus {
	filter := false

	if allowlistPaths, ok := b.allowlist[request.Bucket]; ok {
		for _, allowed := range allowlistPaths {
			if strings.HasPrefix(request.Key[0], allowed) {
				filter = true
				break
			}
		}
	}

	if request.Size > b.sizeLimit {
		filter = true
	}

	switch filter {
	case true:
		return entities.Abort
	default:
		return entities.NextJob
	}
}
