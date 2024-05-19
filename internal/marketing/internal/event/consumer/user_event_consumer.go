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
	"time"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

type RegistrationEventConsumer struct {
	svc       service.Service
	consumer  mq.Consumer
	endAtDate time.Time
	logger    *elog.Component
}

func NewRegistrationEventConsumer(svc service.Service, q mq.MQ) (*RegistrationEventConsumer, error) {
	const groupID = "marketing-user"
	consumer, err := q.Consumer(event.UserRegistrationEventName, groupID)
	if err != nil {
		return nil, err
	}
	return &RegistrationEventConsumer{
		svc:       svc,
		consumer:  consumer,
		endAtDate: time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC),
		logger:    elog.DefaultLogger,
	}, nil
}

// Start 后面要考虑借助 ctx 来优雅退出
func (c *RegistrationEventConsumer) Start(ctx context.Context) {
	go func() {
		for {
			er := c.Consume(ctx)
			if er != nil {
				c.logger.Error("消费注册事件失败", elog.FieldErr(er))
			}
		}
	}()
}

func (c *RegistrationEventConsumer) Consume(ctx context.Context) error {
	msg, err := c.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}
	var evt event.RegistrationEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}
	return c.svc.ExecuteUserRegistrationActivity(ctx, domain.UserRegistrationActivity{Uid: evt.Uid})
}
