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
	"sync"
	"time"

	"github.com/ecodeclub/ekit/retry"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/mq-api/kafka"
	"github.com/gotomicro/ego/core/econf"
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
	type Config struct {
		Network   string   `yaml:"network"`
		Addresses []string `yaml:"addresses"`
	}
	var cfg Config
	econf.Set("kafka.network", "tcp")
	econf.Set("kafka.addresses", []string{"localhost:9092"})
	err := econf.UnmarshalKey("kafka", &cfg)
	if err != nil {
		return nil, err
	}
	qq, err := kafka.NewMQ(cfg.Network, cfg.Addresses)
	if err != nil {
		return nil, err
	}
	return qq, nil
}
