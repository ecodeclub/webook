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
	"log"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit/internal/domain"
	"github.com/ecodeclub/webook/internal/credit/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

type CreditIncreaseConsumer struct {
	svc      service.Service
	consumer mq.Consumer
	logger   *elog.Component
}

func NewCreditIncreaseConsumer(svc service.Service, q mq.MQ) (*CreditIncreaseConsumer, error) {
	groupID := "credit"
	consumer, err := q.Consumer(creditIncreaseEvents, groupID)
	if err != nil {
		return nil, err
	}
	return &CreditIncreaseConsumer{
		svc:      svc,
		consumer: consumer,
		logger:   elog.DefaultLogger,
	}, nil
}

// Start 后面要考虑借助 ctx 来优雅退出
func (c *CreditIncreaseConsumer) Start(ctx context.Context) {
	go func() {
		for {
			err := c.Consume(ctx)
			if err != nil {
				c.logger.Error("消费积分事件失败", elog.FieldErr(err))
			}
		}
	}()
}

func (c *CreditIncreaseConsumer) Consume(ctx context.Context) error {
	msg, err := c.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}

	var evt CreditIncreaseEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	err = c.svc.AddCredits(ctx, domain.Credit{
		Uid:          evt.Uid,
		ChangeAmount: evt.Amount,
		Logs: []domain.CreditLog{
			{
				Key:    evt.Key,
				BizId:  evt.BizId,
				Biz:    evt.Biz,
				Action: evt.Action,
			},
		},
	})

	if err != nil {
		c.logger.Error("变更积分失败",
			elog.FieldErr(err),
			elog.Any("消息体", evt),
		)
	}
	log.Printf("Consumer evt = %#v\n", evt)
	return nil
}

func (c *CreditIncreaseConsumer) Stop(_ context.Context) error {
	return c.consumer.Close()
}
