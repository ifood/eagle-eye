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

package out

import (
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/logging"
	"encoding/json"
)

type CacheIndividualRepository struct {
	cache  out.Cache
	logger logging.Logger
}

func NewCacheIndividualRepository(cache out.Cache, logger logging.Logger) *CacheIndividualRepository {
	return &CacheIndividualRepository{cache: cache, logger: logger}
}

func (c *CacheIndividualRepository) Save(result entities.ScanResult) error {
	jsonResult, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return c.cache.Set(result.ScanID, string(jsonResult), resultTTL)
}

func (c *CacheIndividualRepository) Get(scanID string) (entities.ScanResult, error) {
	jsonResult, err := c.cache.Get(scanID)
	if err != nil {
		c.logger.Errorw("Failed to obtain value for key.", "error", err, "scanId", scanID)
		return entities.NewScanResult(""), err
	}

	var result entities.ScanResult
	if err := json.Unmarshal([]byte(jsonResult), &result); err != nil {
		return entities.NewScanResult(""), err
	}

	return result, nil
}
