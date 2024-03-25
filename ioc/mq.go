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

package ioc

import (
	"context"
	"fmt"
	"time"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/mq-api/kafka"
	"github.com/gotomicro/ego/core/econf"
)

func InitMQ() mq.MQ {
	type Config struct {
		Network   string   `yaml:"network"`
		Addresses []string `yaml:"addresses"`
		Topics    []struct {
			Name       string `yaml:"name"`
			Partitions int    `yaml:"partitions"`
		} `yaml:"topics"`
	}

	var cfg Config
	err := econf.UnmarshalKey("kafka", &cfg)
	if err != nil {
		panic(err)
	}

	q, err := kafka.NewMQ(cfg.Network, cfg.Addresses)
	if err != nil {
		panic(err)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelFunc()
	for i := 0; i < len(cfg.Topics); i++ {
		if e := q.CreateTopic(ctx, cfg.Topics[i].Name, cfg.Topics[i].Partitions); e != nil {
			panic(fmt.Sprintf("创建Topic失败: %s : Topic = %s, Partitions = %d", e.Error(), cfg.Topics[i].Name, cfg.Topics[i].Partitions))
		}
	}
	return q
}
