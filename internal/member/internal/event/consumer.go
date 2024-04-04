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
)

type MQConsumer struct {
	svc         service.Service
	consumer    mq.Consumer
	startAtFunc func() int64
	endAtFunc   func() int64
}

func NewMQConsumer(svc service.Service, consumer mq.Consumer, startAtFunc func() int64, endAtFunc func() int64) *MQConsumer {
	return &MQConsumer{svc: svc, consumer: consumer, startAtFunc: startAtFunc, endAtFunc: endAtFunc}
}

func (c *MQConsumer) ConsumeRegistrationEvent(ctx context.Context) error {
	msg, err := c.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}

	var evt RegistrationEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	_, err = c.svc.GetMembershipInfo(ctx, evt.UserID)
	if err == nil {
		return fmt.Errorf("用户会员记录已存在")
	}

	startAt := c.startAtFunc()
	endAt := c.endAtFunc()
	if endAt <= startAt {
		return fmt.Errorf("超过注册优惠截止日期")
	}

	_, err = c.svc.CreateNewMembership(ctx, domain.Member{
		UserID:  evt.UserID,
		StartAt: c.startAtFunc(),
		EndAt:   c.endAtFunc(),
		Status:  domain.MemberStatusActive,
	})
	return err
}
