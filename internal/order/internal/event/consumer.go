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
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/gotomicro/ego/core/elog"
)

type PaymentConsumer struct {
	svc                service.Service
	consumer           mq.Consumer
	orderEventProducer OrderEventProducer
	logger             *elog.Component
}

func NewPaymentConsumer(svc service.Service, p OrderEventProducer, q mq.MQ) (*PaymentConsumer, error) {
	const groupID = "order"
	consumer, err := q.Consumer(paymentEventName, groupID)
	if err != nil {
		return nil, err
	}
	return &PaymentConsumer{
		svc:                svc,
		consumer:           consumer,
		orderEventProducer: p,
		logger:             elog.DefaultLogger,
	}, nil
}

func (c *PaymentConsumer) Start(ctx context.Context) {
	go func() {
		for {
			er := c.Consume(ctx)
			if er != nil {
				c.logger.Error("消费完成订单事件失败", elog.FieldErr(er))
			}
		}
	}()
}

func (c *PaymentConsumer) Consume(ctx context.Context) error {
	msg, err := c.consumer.Consume(ctx)
	if err != nil {
		return fmt.Errorf("获取消息失败: %w", err)
	}

	var evt PaymentEvent
	err = json.Unmarshal(msg.Value, &evt)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	if evt.Status == uint8(payment.StatusPaidSuccess) {
		err = c.svc.SucceedOrder(ctx, evt.PayerID, evt.OrderSN)
		if err != nil {
			c.logger.Warn("设置订单'支付成功'状态失败",
				elog.FieldErr(err),
				elog.Any("event", evt),
			)
			return err
		}
		return c.sendOrderEvent(ctx, evt)
	} else if evt.Status == uint8(payment.StatusPaidFailed) {
		err = c.svc.FailOrder(ctx, evt.PayerID, evt.OrderSN)
		if err != nil {
			c.logger.Warn("设置订单'支付失败'状态失败",
				elog.FieldErr(err),
				elog.Any("event", evt),
			)
		}
		return err
	} else {
		return fmt.Errorf("未知支付状态: %d", evt.Status)
	}
}

func (c *PaymentConsumer) sendOrderEvent(ctx context.Context, p PaymentEvent) error {
	order, err := c.svc.FindUserVisibleOrderByUIDAndSN(ctx, p.PayerID, p.OrderSN)
	if err != nil {
		c.logger.Warn("发送'订单完成事件'失败",
			elog.FieldErr(err),
			elog.Any("event", p),
		)
		return err
	}
	spus := make([]SPU, 0, len(order.Items))
	for _, item := range order.Items {
		spus = append(spus, SPU{
			ID:        item.SPU.ID,
			Category0: item.SPU.Category0,
			Category1: item.SPU.Category1,
		})
	}
	evt := OrderEvent{
		OrderSN: order.SN,
		BuyerID: order.BuyerID,
		SPUs:    spus,
	}
	err = c.orderEventProducer.Produce(ctx, evt)
	if err != nil {
		c.logger.Warn("发送'订单完成事件'失败",
			elog.FieldErr(err),
			elog.Any("event", evt),
		)
	}
	return err
}
