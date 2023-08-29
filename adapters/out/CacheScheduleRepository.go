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
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	maxTimeWait = time.Hour * 24
)

type CacheScheduleScanRepository struct {
	cache    out.Cache
	listname string
}

func NewCacheScheduleScanRepository[K any](cache out.Cache, listname string) *CacheScheduleScanRepository {
	c := CacheScheduleScanRepository{cache: cache, listname: listname}
	return &c
}

func (c *CacheScheduleScanRepository) Add(key string, item entities.ScheduleItem) error {
	if err := c.cache.ZAddLex(c.listname, []string{fmt.Sprintf("%d:%s", time.Now().Unix(), key)}); err != nil {
		return fmt.Errorf("failed to save remote query id on redis. err: %w", err)
	}

	jsonObject, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("failed to marshall object. err: %w", err)
	}

	if err := c.cache.Set(key, string(jsonObject), maxTimeWait); err != nil {
		return fmt.Errorf("failed to save mapping between scan id and bucket. err: %w", err)
	}

	return nil
}

func (c *CacheScheduleScanRepository) GetUntil(limit time.Time) ([]entities.ScheduleItem, []error) {
	timePrefixedIDs, err := c.cache.ZGetAndRemLex(c.listname, "-", fmt.Sprintf("(%d:", limit.Unix()))
	if err != nil {
		return nil, []error{fmt.Errorf("failed to obtain scan ids. err: %w", err)}
	}

	var items []entities.ScheduleItem
	var errors []error

	for _, timePrefixedID := range timePrefixedIDs {
		const expectedTokensByFormat = 2
		tokenizedID := strings.Split(timePrefixedID, ":")

		if len(tokenizedID) != expectedTokensByFormat {
			errors = append(errors, fmt.Errorf("handle id does not have the proper format of <timestamp>:<scanid>. scanid: %s", timePrefixedID))
			continue
		}

		scanID := tokenizedID[1]
		value, err := c.cache.Get(fmt.Sprintf("vt-%s", scanID))

		if err != nil {
			errors = append(errors, fmt.Errorf("could not obtain key for scanid. err: %w scanid: %s", err, timePrefixedID))
			continue
		}

		var item entities.ScheduleItem
		err = json.NewDecoder(strings.NewReader(value)).Decode(&item)

		if err != nil {
			errors = append(errors, fmt.Errorf("could not unmarshal data. err: %w scanid: %s", err, scanID))
			continue
		}

		items = append(items, item)
	}

	return items, errors
}
