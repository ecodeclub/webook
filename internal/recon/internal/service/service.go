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
			case payment.StatusUnpaid, payment.StatusProcessing:
				err3 := s.handleUnpaidAndProcessingStatus(ctx, o, pmt)
				if err3 != nil {
					s.l.Warn("设置支付状态失败",
						elog.FieldErr(err3),
						elog.Any("payment", pmt),
					)
				}
			case payment.StatusPaidSuccess, payment.StatusPaidFailed:
				err4 := s.handlePaidSuccessAndFailedStatus(ctx, o, pmt)
				if err4 != nil {
					s.l.Warn("设置支付状态失败",
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
扫描数据库中 30 分钟之前处于“支付中”状态的订单，通过订单找到 pmt
1. 如果pmt = “支付成功”, 确认扣减积分并同步调用订单模块方法修改订单状态为“支付成功”
2. 如果pmt = “支付失败”, 取消扣减积分并同步调用订单模块方法修改订单状态为“支付失败”
3. 如果pmt = “未支付”, 同步调用支付模块及订单模块方法修改状态为“支付失败”, 此时无需释放积分.
4. 如果pmt = “支付中”, 同步调用支付模块及订单模块方法修改状态为“支付失败”, 此时需要释放积分.
   注意: 当处于“支付中”状态时，用户可能正在扫码支付，也可能微信那边回调尚未过来。即pmt事实上可能支付了,也可能没有支付。
*/

func (s *service) handleUnpaidAndProcessingStatus(ctx context.Context, o order.Order, pmt payment.Payment) error {
	err := s.paymentSvc.SetPaymentStatusPaidFailed(ctx, &pmt)
	if err != nil {
		return err
	}
	return s.handlePaidSuccessAndFailedStatus(ctx, o, pmt)
}

func (s *service) handlePaidSuccessAndFailedStatus(ctx context.Context, o order.Order, pmt payment.Payment) error {

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
		return nil
	}
}
