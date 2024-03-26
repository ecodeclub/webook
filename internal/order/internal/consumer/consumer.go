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
	"github.com/ecodeclub/webook/internal/order/internal/service"
	"golang.org/x/sync/errgroup"
)

type CompleteOrderConsumer struct {
	svc       service.Service
	consumers []mq.Consumer
}

func NewCompleteOrderConsumer(svc service.Service, consumers []mq.Consumer) *CompleteOrderConsumer {
	return &CompleteOrderConsumer{
		svc:       svc,
		consumers: consumers,
	}
}

func (o *CompleteOrderConsumer) Consume() error {

	messageChan := make(chan *mq.Message)

	var eg errgroup.Group
	for _, c := range o.consumers {
		c := c
		eg.Go(func() error {
			consumeChan, err := c.ConsumeChan(context.Background())
			if err != nil {
				return err
			}
			for msg := range consumeChan {
				messageChan <- msg
			}
			return nil
		})
	}

	eg.Go(func() error {
		return o.completeOrders(messageChan)
	})

	return eg.Wait()
}

func (o *CompleteOrderConsumer) completeOrders(messageChan chan *mq.Message) error {

	type CompleteOrderReq struct {
		OrderSN string `json:"sn"`
		BuyerID int64  `json:"buyerId"`
	}

	for msg := range messageChan {

		var req CompleteOrderReq
		err := json.Unmarshal(msg.Value, &req)
		if err != nil {
			return fmt.Errorf("解析消息体失败: %w", err)
		}

		ctx, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
		order, err := o.svc.FindOrder(ctx, req.OrderSN, req.BuyerID)
		if err != nil {
			cancelFunc()
			return fmt.Errorf("订单未找到: %w", err)
		}

		err = o.svc.CompleteOrder(ctx, order)
		if err != nil {
			cancelFunc()
			return fmt.Errorf("完成订单失败: %w", err)
		}

		cancelFunc()
	}

	return nil
}
