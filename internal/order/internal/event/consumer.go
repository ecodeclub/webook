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
	"github.com/ecodeclub/webook/internal/order/internal/service"
	"github.com/gotomicro/ego/core/elog"
)

type CompleteOrderConsumer struct {
	svc      service.Service
	consumer mq.Consumer
	logger   *elog.Component
}

func NewCompleteOrderConsumer(svc service.Service, q mq.MQ) (*CompleteOrderConsumer, error) {
	const groupID = "order"
	consumer, err := q.Consumer(orderCompleteEvents, groupID)
	if err != nil {
		return nil, err
	}
	return &CompleteOrderConsumer{
		svc:      svc,
		consumer: consumer,
		logger:   elog.DefaultLogger,
	}, nil
}

func (c *CompleteOrderConsumer) Start(ctx context.Context) {
	go func() {
		for {
			er := c.Consume(ctx)
			if er != nil {
				c.logger.Error("消费完成订单事件失败", elog.FieldErr(er))
			}
		}
	}()
}

func (c *CompleteOrderConsumer) Consume(ctx context.Context) error {
	msg, err := c.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}

	var evt CompleteOrderEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}
	// 收到该消息表示用户支付成功,所以不管订单当前的状态是什么都要设置为“已支付完成”
	order, err := c.svc.FindUserVisibleOrderByUIDAndSN(ctx, evt.BuyerID, evt.OrderSN)
	if err != nil {
		c.logger.Error("订单未找到",
			elog.FieldErr(err),
			elog.String("order_sn", evt.OrderSN),
			elog.Int64("buyer_id", evt.BuyerID),
		)
		return fmt.Errorf("订单未找到: %w", err)
	}

	err = c.svc.CompleteOrder(ctx, order.BuyerID, order.ID)
	if err != nil {
		c.logger.Warn("完成订单失败",
			elog.FieldErr(err),
			elog.Int64("order_id", order.ID),
			elog.Int64("buyer_id", order.BuyerID))
	}
	return err
}
