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
	"errors"
	"fmt"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member/internal/domain"
	"github.com/ecodeclub/webook/internal/member/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

type MemberEventConsumer struct {
	svc      service.Service
	consumer mq.Consumer
	logger   *elog.Component
}

func NewMemberEventConsumer(svc service.Service, q mq.MQ) (*MemberEventConsumer, error) {
	groupID := "member"
	consumer, err := q.Consumer(memberUpdateEvents, groupID)
	if err != nil {
		return nil, err
	}
	return &MemberEventConsumer{
		svc:      svc,
		consumer: consumer,
		logger:   elog.DefaultLogger}, nil
}

// Start 后面要考虑借助 ctx 来优雅退出
func (c *MemberEventConsumer) Start(ctx context.Context) {
	go func() {
		for {
			er := c.Consume(ctx)
			if er != nil {
				c.logger.Error("消费会员事件失败", elog.FieldErr(er))
			}
		}
	}()
}

func (c *MemberEventConsumer) Consume(ctx context.Context) error {
	msg, err := c.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}

	var evt MemberEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	err = c.svc.ActivateMembership(ctx, domain.Member{
		Uid: evt.Uid,
		Records: []domain.MemberRecord{
			{
				Key:   evt.Key,
				Days:  evt.Days,
				Biz:   evt.Biz,
				BizId: evt.BizId,
				Desc:  evt.Action,
			},
		},
	})

	if err != nil {
		if errors.Is(err, service.ErrUpdateMemberFailed) {
			// retry
		}

		if errors.Is(err, service.ErrDuplicatedMemberRecord) {
			c.logger.Warn("重复消费",
				elog.FieldErr(err),
				elog.Any("MemberEvent", evt),
			)
		}
		// 其他错误
		c.logger.Error("创建/更新会员相关记录失败",
			elog.FieldErr(err),
			elog.Int64("uid", evt.Uid),
		)
	}
	return err
}
