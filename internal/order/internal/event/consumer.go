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
	svc      service.Service
	consumer mq.Consumer
	logger   *elog.Component
}

func NewPaymentConsumer(svc service.Service, q mq.MQ) (*PaymentConsumer, error) {
	const groupID = "order"
	consumer, err := q.Consumer(paymentEvents, groupID)
	if err != nil {
		return nil, err
	}
	return &PaymentConsumer{
		svc:      svc,
		consumer: consumer,
		logger:   elog.DefaultLogger,
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

	var warnMessage string
	if evt.Status == uint8(payment.StatusPaidSuccess) {
		err = c.svc.SucceedOrder(ctx, evt.PayerID, evt.OrderSN)
		warnMessage = "设置订单'支付成功'状态失败"
	} else if evt.Status == uint8(payment.StatusPaidFailed) {
		err = c.svc.FailOrder(ctx, evt.PayerID, evt.OrderSN)
		warnMessage = "设置订单'支付失败'状态失败"
	} else {
		return fmt.Errorf("未知支付状态: %d", evt.Status)
	}

	if err != nil {
		c.logger.Warn(warnMessage,
			elog.FieldErr(err),
			elog.Any("event", evt),
		)
	}
	return err

}
