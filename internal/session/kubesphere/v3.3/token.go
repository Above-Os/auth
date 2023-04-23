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
	"os"
	"strconv"
	"time"

	"k8s.io/klog/v2"
)

var (
	StorageHost     string = "ks-api-redis.kubesphere-system"
	StoragePort     int    = 6379
	StoragePassword string
	StorageDB       int = 0
)

func init() {
	for e, v := range map[string]interface{}{
		"KS_TOKEN_HOST":     &StorageHost,
		"KS_TOKEN_PORT":     &StoragePort,
		"KS_TOKEN_PASSWORD": &StoragePassword,
		"KS_TOKEN_DB":       &StorageDB,
	} {
		ev := os.Getenv(e)

		var err error

		switch raw := v.(type) {
		case *string:
			*raw = ev
		case *int:
			*raw, err = strconv.Atoi(ev)
			if err != nil {
				klog.Error("read env ", e, " - ", ev, " error, ", err)
			}
		}
	}
}

type Operator struct {
	client Interface

	ctx   context.Context
	Close context.CancelFunc
}

func NewTokenOperator() (*Operator, error) {
	options := &DynamicOptions{
		"host":     StorageHost,
		"port":     StoragePort,
		"password": StoragePassword,
		"db":       StorageDB,
	}

	ctx, close := context.WithCancel(context.Background())

	client, err := NewRedisClient(options, ctx, ctx.Done())

	if err != nil {
		close()
		return nil, err
	}

	return &Operator{
		client: client,
		ctx:    ctx,
		Close:  close,
	}, nil
}

func (o *Operator) RestoreToken(username, token string, duration time.Duration) {
	key := fmt.Sprintf("kubesphere:user:%s:token:%s", username, token)
	if exist, err := o.client.Exists(key); err != nil {
		klog.Error("validate token error, ", err, " : ", key)
	} else if !exist {
		if err := o.client.Set(key, token, duration); err != nil {
			klog.Error("set token error, ", err)
		}
	}
}
