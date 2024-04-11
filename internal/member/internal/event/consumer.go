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

package event

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member/internal/domain"
	"github.com/ecodeclub/webook/internal/member/internal/service"
	"github.com/gotomicro/ego/core/elog"
	"github.com/lithammer/shortuuid/v4"
)

type RegistrationEventConsumer struct {
	svc       service.Service
	consumer  mq.Consumer
	endAtDate time.Time
	logger    *elog.Component
}

func NewRegistrationEventConsumer(svc service.Service,
	q mq.MQ) (*RegistrationEventConsumer, error) {
	const groupID = "member"
	consumer, err := q.Consumer(userRegistrationEvents, groupID)
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

	var evt RegistrationEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	err = c.svc.CreateNewMembership(ctx, domain.Member{
		Uid: evt.Uid,
		Records: []domain.MemberRecord{
			{
				Key:   shortuuid.New(),
				Biz:   1,
				BizId: evt.Uid,
				Desc:  "注册福利",
				Days:  uint64(time.Until(c.endAtDate) / (24 * time.Hour)),
			},
		},
	})
	if err != nil {
		c.logger.Error("创建会员记录失败",
			elog.FieldErr(err),
			elog.Int64("uid", evt.Uid),
		)
	}
	return err
}
