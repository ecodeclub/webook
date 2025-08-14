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
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/notification/wechat/consumer"
	"github.com/gotomicro/ego/core/econf"
)

func initMQConsumers(q mq.MQ) []Consumer {
	return []Consumer{
		initWechatRobotEventConsumer(q),
	}
}

func initWechatRobotEventConsumer(q mq.MQ) *consumer.WechatRobotEventConsumer {
	var cfg consumer.WechatRobotConfig
	err := econf.UnmarshalKey("qywechat", &cfg)
	if err != nil {
		panic(err)
	}
	res, err := consumer.NewWechatRobotEventConsumer(q, &cfg)
	if err != nil {
		panic(err)
	}
	return res
}
