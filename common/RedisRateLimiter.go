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

package common

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redis_rate/v9"
)

//go:generate go run -mod=mod github.com/golang/mock/mockgen -destination=../mocks/mock_rate_limiter.go -package=mocks -source=RedisRateLimiter.go
type RateLimiter interface {
	IsRequestAllowed() bool
}

type RateLimitConfig struct {
	Hour   int
	Minute int
	Key    string
}

type RedisRateLimiter struct {
	config  RateLimitConfig
	limiter *redis_rate.Limiter
}

func NewRateLimiter(url, password string, useTLS bool, config RateLimitConfig) *RedisRateLimiter {
	rateLimiter := RedisRateLimiter{}

	options := redis.Options{
		Addr:     url,
		Password: password,
		DB:       0, // use default DB
	}

	if useTLS {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	rateLimiter.config = config
	rateLimiter.limiter = redis_rate.NewLimiter(redis.NewClient(&options))

	return &rateLimiter
}

func (r *RedisRateLimiter) Init(url, password string, useTLS bool, config RateLimitConfig) {
	options := redis.Options{
		Addr:     url,
		Password: password,
		DB:       0, // use default DB
	}

	if useTLS {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	r.config = config
	r.limiter = redis_rate.NewLimiter(redis.NewClient(&options))
}

func (r *RedisRateLimiter) IsRequestAllowed() bool {
	ctx := context.Background()

	if r.config.Minute != 0 {
		res, err := r.limiter.Allow(ctx, fmt.Sprintf("Minute-%s", r.config.Key), redis_rate.PerMinute(r.config.Minute))
		if err != nil || res.Allowed == 0 {
			return false
		}
	}

	if r.config.Hour != 0 {
		res, err := r.limiter.Allow(ctx, fmt.Sprintf("Hour-%s", r.config.Key), redis_rate.PerHour(r.config.Hour))
		if err != nil || res.Allowed == 0 {
			return false
		}
	}

	return true
}
