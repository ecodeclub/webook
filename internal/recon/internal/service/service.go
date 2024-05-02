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

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ecodeclub/ekit/retry"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/gotomicro/ego/core/elog"
)

type Service interface {
	Reconcile(ctx context.Context, offset, limit int, ctime int64) error
}

type service struct {
	orderSvc        order.Service
	paymentSvc      payment.Service
	creditSvc       credit.Service
	initialInterval time.Duration
	maxInterval     time.Duration
	maxRetries      int32
	l               *elog.Component
}

func NewService(orderSvc order.Service,
	paymentSvc payment.Service,
	creditSvc credit.Service, initialInterval time.Duration, maxInterval time.Duration, maxRetries int32) *service {
	return &service{orderSvc: orderSvc,
		paymentSvc:      paymentSvc,
		creditSvc:       creditSvc,
		initialInterval: initialInterval, maxInterval: maxInterval, maxRetries: maxRetries,
		l: elog.DefaultLogger}
}

func (s *service) Reconcile(ctx context.Context, offset, limit int, ctime int64) error {
	for {

		orders, total, err := s.orderSvc.FindTimeoutOrders(ctx, offset, limit, ctime)
		if err != nil {
			return fmt.Errorf("查找超时订单失败: %w", err)
		}

		for _, o := range orders {
			pmt, err2 := s.paymentSvc.FindPaymentByID(ctx, o.Payment.ID)
			if err2 != nil {
				s.l.Warn("通过超时订单查找支付失败",
					elog.FieldErr(err2),
					elog.Any("order", o),
				)
				continue
			}

			switch pmt.Status {
			case payment.StatusUnpaid:
			case payment.StatusProcessing:
				err3 := s.handleUnpaidAndProcessingStatus(ctx, o, pmt)
				if err3 != nil {
					s.l.Warn("设置支付失败",
						elog.FieldErr(err3),
						elog.Any("payment", pmt),
					)
				}
			case payment.StatusPaidSuccess:
			case payment.StatusPaidFailed:
				err4 := s.handlePaidSuccessAndPaidFailedStatus(ctx, o, pmt)
				if err4 != nil {
					s.l.Warn("处理支付失败",
						elog.FieldErr(err4),
						elog.Any("payment", pmt),
					)
				}
			}
		}

		if len(orders) < limit {
			return nil
		}

		if int64(limit) >= total {
			return nil
		}
	}
}

/*
扫描数据库中 30 分钟之前处于 PAYING 状态的订单，找到 pmt。（这个从 pmt 这边发起任务会比较容易）
1. 如果pmt = “支付成功”, 发送“支付成功”消息,订单模块消费者会将订单状态更新为“支付成功”
2. 如果pmt = “支付失败”, 发送“支付失败”消息,订单模块消费者会将订单状态更新为“支付失败”
3. 如果pmt= “未支付” INT, 直接调用PaymentSvc和OrderSvc修改状态为“支付失败”, 此时无需释放积分.
4. 如果pmt =支付中, 调用PaymentSvc和OrderSvc修改状态为“支付失败”, 此时需要释放积分.
(这里可能与下方“支付模块”的定时任务冲突,
需要确定时间间隔,来保证pmt=INT/Paying是支付模块定时任务处理过后,剩下来的,
这样也就可以省去支付模块的定时任务1)

如果 pmt 已经是终结状态（支付成功/支付失败），更新对应的 order 状态，扣减/释放积分。
如果 pmt 处于 INIT，paying 状态, 订单状态直接将 支付置为失败，订单失败，释放积分。

*/

func (s *service) handleUnpaidAndProcessingStatus(ctx context.Context, o order.Order, pmt payment.Payment) error {
	err := s.paymentSvc.SetPaymentStatusPaidFailed(ctx, &pmt)
	if err != nil {
		return err
	}
	return s.handlePaidSuccessAndPaidFailedStatus(ctx, o, pmt)
}

func (s *service) handlePaidSuccessAndPaidFailedStatus(ctx context.Context, o order.Order, pmt payment.Payment) error {

	strategy, er := retry.NewExponentialBackoffRetryStrategy(s.initialInterval, s.maxInterval, s.maxRetries)
	if er != nil {
		return er
	}

	var err error
	for {

		d, ok := strategy.Next()
		if !ok {
			s.l.Warn("处理支付成功及支付失败超过最大重试次数",
				elog.Any("order", o),
				elog.Any("payment", pmt),
			)
			return fmt.Errorf("超过最大重试次数")
		}

		// 在混合支付的时候需要对积分支付进行额外处理
		// 支付成功 —— 确认扣减积分+状态更新
		// 支付失败 —— 取消扣见积分+状态更新
		err = s.paymentSvc.HandleCreditCallback(ctx, pmt)
		if err != nil {
			time.Sleep(d)
			continue
		}

		if pmt.Status == payment.StatusPaidSuccess {
			err = s.orderSvc.SucceedOrder(ctx, o.BuyerID, o.SN)
		} else {
			err = s.orderSvc.FailOrder(ctx, o.BuyerID, o.SN)
		}
		if err != nil {
			s.l.Warn("根据支付状态更新订单状态失败",
				elog.FieldErr(err),
				elog.Any("order", o),
				elog.Any("payment", pmt),
			)
			time.Sleep(d)
			continue
		}

		oo, err := s.orderSvc.FindUserVisibleOrderByUIDAndSN(ctx, o.BuyerID, o.SN)
		if err != nil {
			s.l.Warn("轮训订单状态失败",
				elog.FieldErr(err),
				elog.Any("order", oo),
			)
			continue
		}

		if (oo.Status == order.StatusSuccess && pmt.Status == payment.StatusPaidSuccess) ||
			(oo.Status == order.StatusFailed && pmt.Status == payment.StatusPaidFailed) {
			return nil
		}
	}
}
