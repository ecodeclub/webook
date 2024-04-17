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

package credit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ecodeclub/ekit/retry"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/events"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/gotomicro/ego/core/elog"
	"github.com/lithammer/shortuuid/v4"
)

var (
	ErrExceedTheMaximumNumberOfRetries = errors.New("超过最大重试次数")
)

type PaymentService struct {
	svc            credit.Service
	repo           repository.PaymentRepository
	producer       events.Producer
	paymentDDLFunc func() int64
	snGenerator    *sequencenumber.Generator
	l              *elog.Component

	initialInterval time.Duration
	maxInterval     time.Duration
	maxRetries      int32
}

func NewCreditPaymentService(svc credit.Service,
	repo repository.PaymentRepository,
	producer events.Producer,
	paymentDDLFunc func() int64,
	snGenerator *sequencenumber.Generator,
	l *elog.Component,
) *PaymentService {
	return &PaymentService{
		svc:             svc,
		repo:            repo,
		producer:        producer,
		paymentDDLFunc:  paymentDDLFunc,
		snGenerator:     snGenerator,
		l:               l,
		initialInterval: 100 * time.Millisecond,
		maxInterval:     1 * time.Second,
		maxRetries:      3,
	}
}

// Pay 直接支付 先立即支付然后立即处理回调
func (p *PaymentService) Pay(ctx context.Context, pmt domain.Payment) (domain.Payment, error) {
	createdPayment, err := p.Prepay(ctx, pmt)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("积分支付失败: %w", err)
	}

	paidAt := time.Now().UnixMilli()
	createdPayment.PaidAt = paidAt
	createdPayment.Status = domain.PaymentStatusPaid

	// todo: 在订单模块添加一个状态, 已失败
	err2 := p.HandleCallback(ctx, createdPayment)
	if err2 != nil {

		// todo: 是否需要发送“支付失败”消息?
		// 如果发送, 这边也应该同步修改支付记录的状体为支付失败
		err3 := p.producer.ProducePaymentEvent(ctx, events.PaymentEvent{
			OrderSN: pmt.OrderSN,
			Status:  domain.PaymentStatusFailed,
		})
		if err3 != nil {
			p.l.Error("发送积分支付成功事件失败",
				elog.FieldErr(err3),
				elog.String("order_sn", pmt.OrderSN),
			)
		}
		return domain.Payment{}, fmt.Errorf("积分支付失败: %w", err2)
	}

	// todo: 发送“支付成功”消息
	// {user_id, “购买商品”, order_id, order_sn, paidAmount}
	err4 := p.producer.ProducePaymentEvent(ctx, events.PaymentEvent{
		OrderSN: pmt.OrderSN,
		Status:  domain.PaymentStatusPaid,
	})
	if err4 != nil {
		p.l.Error("发送积分支付成功事件失败",
			elog.FieldErr(err4),
			elog.String("order_sn", pmt.OrderSN),
		)
	}

	return createdPayment, nil
}

// Prepay 预支付
func (p *PaymentService) Prepay(ctx context.Context, pmt domain.Payment) (domain.Payment, error) {

	r, ok := slice.Find(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeCredit
	})
	if !ok || r.Amount == 0 {
		return domain.Payment{}, fmt.Errorf("缺少积分支付金额信息")
	}

	paymentSN, err := p.snGenerator.Generate(pmt.PayerID)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("生成支付序列号失败: %w", err)
	}
	pmt.SN = paymentSN

	paymentNO3rd, err := p.tryDeductCredits(ctx, pmt.PayerID, uint64(r.Amount))
	if err != nil {
		return domain.Payment{}, fmt.Errorf("预扣积分失败")
	}

	pmt.PayDDL = p.paymentDDLFunc()
	pmt.Status = domain.PaymentStatusUnpaid
	pmt.Records = []domain.PaymentRecord{
		{
			PaymentNO3rd: strconv.FormatInt(paymentNO3rd, 10),
			Description:  pmt.OrderDescription,
			Channel:      domain.ChannelTypeCredit,
			Amount:       r.Amount,
			Status:       domain.PaymentStatusUnpaid,
		},
	}

	pp, err2 := p.repo.CreatePayment(ctx, pmt)
	if err2 != nil {
		// 取消预扣
		err3 := p.cancelDeductCredits(ctx, pmt.PayerID, paymentNO3rd)
		if err3 != nil {
			return domain.Payment{}, fmt.Errorf("创建支付主记录及积分渠道支付记录失败: %w: %w", err2, err3)
		}
		return domain.Payment{}, fmt.Errorf("创建支付主记录及积分渠道支付记录失败: %w", err2)
	}
	return pp, nil
}

func (p *PaymentService) tryDeductCredits(ctx context.Context, uid int64, amount uint64) (txID int64, err error) {
	strategy, _ := retry.NewExponentialBackoffRetryStrategy(p.initialInterval, p.maxInterval, p.maxRetries)
	for {
		txID, err := p.svc.TryDeductCredits(ctx, credit.Credit{Uid: uid, Logs: []credit.CreditLog{
			{
				Key:          shortuuid.New(),
				ChangeAmount: int64(amount),
				Biz:          "payment",
				BizId:        0,
				Desc:         "",
			},
		}})
		if err == nil {
			return txID, nil
		}
		next, ok := strategy.Next()
		if !ok {
			return 0, fmt.Errorf("预扣积分超时失败: %w", ErrExceedTheMaximumNumberOfRetries)
		}
		time.Sleep(next)
	}
}

func (p *PaymentService) cancelDeductCredits(ctx context.Context, uid, tid int64) error {
	strategy, _ := retry.NewExponentialBackoffRetryStrategy(p.initialInterval, p.maxInterval, p.maxRetries)
	for {
		err := p.svc.CancelDeductCredits(ctx, uid, tid)
		if err == nil {
			return nil
		}
		next, ok := strategy.Next()
		if !ok {
			return fmt.Errorf("取消预扣积分失败: %w", ErrExceedTheMaximumNumberOfRetries)
		}
		time.Sleep(next)
	}
}

// HandleCallback 处理回调
func (p *PaymentService) HandleCallback(ctx context.Context, pmt domain.Payment) error {
	var paymentNO3rd string
	pmt.Records = slice.Map(pmt.Records, func(idx int, src domain.PaymentRecord) domain.PaymentRecord {
		if src.Channel == domain.ChannelTypeCredit {
			src.Status = pmt.Status
			if src.Status == domain.PaymentStatusPaid {
				src.PaidAt = pmt.PaidAt
			}
			paymentNO3rd = src.PaymentNO3rd
		}
		return src
	})

	txID, _ := strconv.ParseInt(paymentNO3rd, 10, 64)
	err := p.confirmDeductCredits(ctx, pmt.PayerID, txID)
	if err != nil {
		err2 := p.cancelDeductCredits(ctx, pmt.PayerID, txID)
		if err2 != nil {
			return fmt.Errorf("积分支付失败: %w: %w", err, err2)
		}
		// todo: 发送“支付失败”消息到消息队列, 通知“订单模块”关闭订单?
		return fmt.Errorf("积分支付失败: %w", err)
	}

	// 更新支付记录
	err3 := p.repo.UpdatePayment(ctx, pmt)
	if err3 != nil {
		// todo: 事务问题, 积分扣减成功, 但是创建支付主记录及积分支付记录失败该怎么办?
		// 记录日志, 人工补偿?
		p.l.Error("积分已扣除但更新支付主记录及积分支付渠道记录失败",
			elog.FieldErr(err3),
			elog.Int64("payment_id", pmt.ID),
			elog.Int64("payer_id", pmt.PayerID),
			elog.String("payment_no_3rd", paymentNO3rd),
		)
		return fmt.Errorf("创建支付主记录及积分渠道支付记录失败: %w", err3)
	}
	return nil
}

func (p *PaymentService) confirmDeductCredits(ctx context.Context, uid, tid int64) error {
	strategy, _ := retry.NewExponentialBackoffRetryStrategy(p.initialInterval, p.maxInterval, p.maxRetries)
	for {

		err := p.svc.ConfirmDeductCredits(ctx, uid, tid)
		if err == nil {
			return nil
		}
		next, ok := strategy.Next()
		if !ok {
			return fmt.Errorf("确认扣减预扣积分失败: %w", ErrExceedTheMaximumNumberOfRetries)
		}
		time.Sleep(next)
	}
}
