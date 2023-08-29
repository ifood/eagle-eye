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
	"eagle-eye/common"
	"eagle-eye/domain/entities"
	"eagle-eye/domain/ports/out"
	"eagle-eye/logging"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"
	"strings"
	"time"
)

const (
	monthlyKeyFormat = "%02d/*"
	dailyKeyFormat   = "%02d/%02d/*"
	bucketKeyFormat  = "%02d/%02d/%s"

	maxSleepForRetry = 30
	lockInterval     = 60 * time.Second
	lockKeyFormat    = "lock-%s"

	resultTTL = 32 * 24 * time.Hour
)

type CacheAggregateRepo struct {
	cache  out.Cache
	logger logging.Logger
}

func NewCacheAggregateRepo(cache out.Cache, logger logging.Logger) *CacheAggregateRepo {
	return &CacheAggregateRepo{cache: cache, logger: logger}
}

func (c *CacheAggregateRepo) Save(result entities.ScanResult) error {
	c.lock(result.Bucket)
	defer c.unlock(result.Bucket)

	jsonResult, err := json.Marshal(result)
	if err != nil {
		return err
	}

	key := c.getItemKey(result.LastUpdate.Day(), int(result.LastUpdate.Month()), result.Bucket)

	return c.cache.Set(key, string(jsonResult), resultTTL)
}

func (c *CacheAggregateRepo) getItemKey(day, month int, bucket string) string {
	return fmt.Sprintf(bucketKeyFormat, month, day, bucket)
}

func (c *CacheAggregateRepo) getDailyKey(day, month int) string {
	return fmt.Sprintf(dailyKeyFormat, month, day)
}

func (c *CacheAggregateRepo) GetByDate(day, month int) (map[string]entities.ScanResult, error) {
	key := c.getDailyKey(day, month)
	return c.listKeys(key)
}

func (c *CacheAggregateRepo) listKeys(key string) (map[string]entities.ScanResult, error) {
	results := make(map[string]entities.ScanResult)

	keys, err := c.cache.List(key)
	if err != nil {
		return nil, fmt.Errorf("error getting keys in redis. %w", err)
	}

	for _, key := range keys {
		bucket := strings.Split(key, "/")[2]
		result, err := c.getSingleScanResult(key, bucket)

		if err != nil {
			c.logger.Errorw("Failed to obtain value for key", "error", err, "key", key)
			continue
		}

		results[bucket] = entities.MergeScanResults(results[bucket], result)
	}

	return results, nil
}

func (c *CacheAggregateRepo) getMonthKey(month int) string {
	return fmt.Sprintf(monthlyKeyFormat, month)
}

func (c *CacheAggregateRepo) GetByMonth(month int) (map[string]entities.ScanResult, error) {
	key := c.getMonthKey(month)
	return c.listKeys(key)
}

func (c *CacheAggregateRepo) GetByBucketAndDate(bucket string, day, month int) (entities.ScanResult, error) {
	c.lock(bucket)
	defer c.unlock(bucket)

	key := c.getItemKey(day, month, bucket)

	return c.getSingleScanResult(key, bucket)
}

func (c *CacheAggregateRepo) getSingleScanResult(key, bucket string) (entities.ScanResult, error) {
	jsonResult, err := c.cache.Get(key)
	if errors.Is(err, redis.Nil) {
		return entities.NewScanResult(bucket), nil
	}

	if err != nil {
		return entities.NewScanResult(bucket), err
	}

	var result entities.ScanResult
	if err := json.Unmarshal([]byte(jsonResult), &result); err != nil {
		return entities.NewScanResult(bucket), err
	}

	return result, nil
}

func (c *CacheAggregateRepo) lock(key string) {
	lockKey := fmt.Sprintf(lockKeyFormat, key)

	// TODO: Replace all http calls with proper library with circuit break and exponential backoff and max attempts.
	// BUG: This can cause the server to become irresponsive because of stalling the pipeline
	// Fix all http comunication globally in another MR.
	for err := c.cache.Lock(lockKey, lockInterval); err != nil; {
		time.Sleep(time.Duration(common.RandInt(maxSleepForRetry)) * time.Second)
	}
}

func (c *CacheAggregateRepo) unlock(key string) {
	lockKey := fmt.Sprintf(lockKeyFormat, key)
	err := c.cache.Unlock(lockKey)

	if err != nil {
		c.logger.Errorw("failed to unlock key for bucket", "error", err, "key", key)
	}
}
