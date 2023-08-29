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

package awsutils

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/bsm/redislock"
	"github.com/go-redis/redis/v9"
)

const (
	minBackoffTime = 20 * time.Millisecond
	maxBackoffTime = 30 * time.Second
	maxLockRetry   = 10
)

type Elasticache struct {
	ctx    context.Context
	rdb    *redis.Client
	locker *redislock.Client
}

func (e *Elasticache) InitRedis(url, password string, useTLS bool) {
	options := redis.Options{
		Addr:     url,
		Password: password,
		DB:       0, // use default DB
	}

	if useTLS {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	e.ctx = context.Background()
	e.rdb = redis.NewClient(&options)
	e.locker = redislock.New(e.rdb)
}

func (e *Elasticache) GetKey(key string) (string, error) {
	return e.rdb.Get(e.ctx, key).Result()
}

func (e *Elasticache) SetKey(key string, value any, expiration time.Duration) error {
	return e.rdb.Set(e.ctx, key, value, expiration).Err()
}

func (e *Elasticache) ListKeys(pattern string) ([]string, error) {
	return e.rdb.Keys(e.ctx, pattern).Result()
}

func (e *Elasticache) Lock(key string, duration time.Duration) (*redislock.Lock, error) {
	return e.locker.Obtain(e.ctx, key, duration, &redislock.Options{
		RetryStrategy: redislock.LimitRetry(redislock.ExponentialBackoff(minBackoffTime, maxBackoffTime), maxLockRetry),
	})
}

func (e *Elasticache) Unlock(lock *redislock.Lock) error {
	if lock != nil {
		return lock.Release(e.ctx)
	}

	return nil
}

func (e *Elasticache) ZAddLex(key string, values []string) error {
	var members []redis.Z
	for _, value := range values {
		members = append(members, redis.Z{
			Score:  0,
			Member: value,
		})
	}

	return e.rdb.ZAdd(e.ctx, key, members...).Err()
}

func (e *Elasticache) ZGetAndRemLex(key, min, max string) ([]string, error) {
	var results []string

	tx := func(tx *redis.Tx) error {
		keys, err := tx.ZRangeByLex(e.ctx, key, &redis.ZRangeBy{Min: min, Max: max}).Result()
		if err != nil {
			return err
		}

		results = append(results, keys...)

		_, err = tx.TxPipelined(e.ctx, func(pipe redis.Pipeliner) error {
			_, err = pipe.ZRemRangeByLex(e.ctx, key, min, max).Result()
			return err
		})

		return err
	}

	err := e.rdb.Watch(e.ctx, tx, key)

	return results, err
}
