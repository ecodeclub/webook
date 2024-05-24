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

package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

type UserRegistrationEventConsumer struct {
	svc      service.Service
	consumer mq.Consumer
	logger   *elog.Component
}

func NewUserRegistrationEventConsumer(svc service.Service, q mq.MQ) (*UserRegistrationEventConsumer, error) {
	const groupID = "marketing-user"
	consumer, err := q.Consumer(event.UserRegistrationEventName, groupID)
	if err != nil {
		return nil, err
	}
	return &UserRegistrationEventConsumer{
		svc:      svc,
		consumer: consumer,
		logger:   elog.DefaultLogger,
	}, nil
}

// Start 后面要考虑借助 ctx 来优雅退出
func (c *UserRegistrationEventConsumer) Start(ctx context.Context) {
	go func() {
		for {
			er := c.Consume(ctx)
			if er != nil {
				c.logger.Error("消费注册事件失败", elog.FieldErr(er))
			}
		}
	}()
}

func (c *UserRegistrationEventConsumer) Consume(ctx context.Context) error {
	msg, err := c.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}
	var evt event.UserRegistrationEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}
	return c.svc.ExecuteUserRegistrationActivity(ctx, domain.UserRegistrationActivity{Uid: evt.Uid, InviterCode: evt.InviterCode})
}
