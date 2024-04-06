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

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member/internal/domain"
	"github.com/ecodeclub/webook/internal/member/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

type RegistrationEventConsumer struct {
	svc         service.Service
	consumer    mq.Consumer
	startAtFunc func() int64
	endAtFunc   func() int64
	logger      *elog.Component
}

func NewRegistrationEventConsumer(svc service.Service, consumer mq.Consumer, startAtFunc func() int64, endAtFunc func() int64) *RegistrationEventConsumer {
	return &RegistrationEventConsumer{
		svc:         svc,
		consumer:    consumer,
		startAtFunc: startAtFunc,
		endAtFunc:   endAtFunc,
		logger:      elog.DefaultLogger}
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

	_, err = c.svc.GetMembershipInfo(ctx, evt.Uid)
	if err == nil {
		return fmt.Errorf("用户会员记录已存在")
	}

	startAt := c.startAtFunc()
	endAt := c.endAtFunc()
	if endAt <= startAt {
		return fmt.Errorf("超过注册优惠截止日期")
	}

	_, err = c.svc.CreateNewMembership(ctx, domain.Member{
		UID:     evt.Uid,
		StartAt: c.startAtFunc(),
		EndAt:   c.endAtFunc(),
	})
	if err != nil {
		c.logger.Error("创建会员记录失败",
			elog.FieldErr(err),
			elog.Int64("uid", evt.Uid),
		)
	}
	return err
}
