// Copyright 2023 bytetrade
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubesphere

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
)

type Interface interface {
	// Keys retrieves all keys match the given pattern.
	Keys(pattern string) ([]string, error)

	// Get retrieves the value of the given key, return error if key doesn't exist.
	Get(key string) (string, error)

	// Set sets the value and living duration of the given key, zero duration means never expire.
	Set(key string, value string, duration time.Duration) error

	// Del deletes the given key, no error returned if the key doesn't exists.
	Del(keys ...string) error

	// Exists checks the existence of a give key.
	Exists(keys ...string) (bool, error)

	// Expires updates object's expiration time, return err if key doesn't exist.
	Expire(key string, duration time.Duration) error
}

type DynamicOptions map[string]interface{}

type redisClient struct {
	client *redis.Client
	ctx    context.Context
}

// redisOptions used to create a redis client.
type redisOptions struct {
	Host     string `json:"host" yaml:"host" mapstructure:"host"`
	Port     int    `json:"port" yaml:"port" mapstructure:"port"`
	Password string `json:"password" yaml:"password" mapstructure:"password"`
	DB       int    `json:"db" yaml:"db" mapstructure:"db"`
}

func NewRedisClient(option *DynamicOptions, ctx context.Context, stopCh <-chan struct{}) (Interface, error) {
	var rOptions redisOptions
	if err := mapstructure.Decode(option, &rOptions); err != nil {
		return nil, err
	}

	var r redisClient

	redisOptions := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", rOptions.Host, rOptions.Port),
		Password: rOptions.Password,
		DB:       rOptions.DB,
	}

	if stopCh == nil {
		klog.Fatalf("no stop channel passed, redis connections will leak.")
	}

	r.client = redis.NewClient(redisOptions)
	r.ctx = ctx

	if err := r.client.Ping(r.ctx).Err(); err != nil {
		r.client.Close()
		return nil, err
	}

	// close redis in case of connection leak.
	if stopCh != nil {
		go func() {
			<-stopCh

			if err := r.client.Close(); err != nil {
				klog.Error(err)
			}
		}()
	}

	return &r, nil
}

func (r *redisClient) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

func (r *redisClient) Keys(pattern string) ([]string, error) {
	return r.client.Keys(r.ctx, pattern).Result()
}

func (r *redisClient) Set(key string, value string, duration time.Duration) error {
	return r.client.Set(r.ctx, key, value, duration).Err()
}

func (r *redisClient) Del(keys ...string) error {
	return r.client.Del(r.ctx, keys...).Err()
}

func (r *redisClient) Exists(keys ...string) (bool, error) {
	existedKeys, err := r.client.Exists(r.ctx, keys...).Result()
	if err != nil {
		return false, err
	}

	return len(keys) == int(existedKeys), nil
}

func (r *redisClient) Expire(key string, duration time.Duration) error {
	return r.client.Expire(r.ctx, key, duration).Err()
}
