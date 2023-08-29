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
	"eagle-eye/pkg/awsutils"
	"fmt"
	"github.com/bsm/redislock"
	"time"
)

type AWSCache struct {
	locks       map[string]*redislock.Lock
	elasticache awsutils.Elasticache
}

func NewCache(url, password string, useTLS bool) *AWSCache {
	elasticache := awsutils.Elasticache{}
	elasticache.InitRedis(url, password, useTLS)

	return &AWSCache{
		elasticache: elasticache,
		locks:       make(map[string]*redislock.Lock),
	}
}

func (a *AWSCache) Get(key string) (string, error) {
	return a.elasticache.GetKey(key)
}

func (a *AWSCache) Set(key string, value interface{}, expiration time.Duration) error {
	return a.elasticache.SetKey(key, value, expiration)
}

func (a *AWSCache) List(pattern string) ([]string, error) {
	return a.elasticache.ListKeys(pattern)
}

func (a *AWSCache) Lock(key string, duration time.Duration) error {
	lock, err := a.elasticache.Lock(key, duration)
	if err == nil {
		a.locks[key] = lock
	}

	return err
}

func (a *AWSCache) Unlock(key string) error {
	if lock, ok := a.locks[key]; ok {
		delete(a.locks, key)
		return a.elasticache.Unlock(lock)
	}

	return fmt.Errorf("lock not found. key %s", key)
}

func (a *AWSCache) ZAddLex(key string, values []string) error {
	return a.elasticache.ZAddLex(key, values)
}

func (a *AWSCache) ZGetAndRemLex(key, min, max string) ([]string, error) {
	return a.elasticache.ZGetAndRemLex(key, min, max)
}
