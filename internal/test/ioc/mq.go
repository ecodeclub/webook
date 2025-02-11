// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testioc

import (
	"context"
	"sync"
	"time"

	"github.com/ecodeclub/ekit/retry"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/mq-api/memory"
)

var (
	q          mq.MQ
	mqInitOnce sync.Once
)

func InitMQ() mq.MQ {
	mqInitOnce.Do(func() {
		const maxInterval = 10 * time.Second
		const maxRetries = 10
		strategy, err := retry.NewExponentialBackoffRetryStrategy(time.Second, maxInterval, maxRetries)
		if err != nil {
			panic(err)
		}
		for {
			q, err = initMQ()
			if err == nil {
				break
			}
			next, ok := strategy.Next()
			if !ok {
				panic("InitMQ 重试失败......")
			}
			time.Sleep(next)
		}
	})
	return q
}

func initMQ() (mq.MQ, error) {
	type Topic struct {
		Name       string `yaml:"name"`
		Partitions int    `yaml:"partitions"`
	}

	topics := []Topic{
		{
			Name:       "payment_events",
			Partitions: 1,
		},
		{
			Name:       "user_registration_events",
			Partitions: 1,
		},
		{
			Name:       "credit_increase_events",
			Partitions: 1,
		},
		{
			Name:       "member_update_events",
			Partitions: 1,
		},
		{
			Name:       "sync_data_to_search",
			Partitions: 1,
		},
		{
			Name:       "order_events",
			Partitions: 1,
		},
		{
			Name:       "interactive_events",
			Partitions: 1,
		},
		{
			Name:       "create_product",
			Partitions: 1,
		},
		{
			Name:       "knowledge_base_upload_topic",
			Partitions: 1,
		},
	}
	// 替换用内存实现，方便测试
	qq := memory.NewMQ()
	for _, t := range topics {
		err := qq.CreateTopic(context.Background(), t.Name, t.Partitions)
		if err != nil {
			return nil, err
		}
	}
	return qq, nil
}
