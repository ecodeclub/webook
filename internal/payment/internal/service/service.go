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

	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
)

type Service interface {
	CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error)
	GetPaymentChannels(ctx context.Context) []domain.PaymentChannel
	FindPaymentByID(ctx context.Context, paymentID int64) (domain.Payment, error)
}

func NewService(repo repository.PaymentRepository, service2 credit.Service) Service {
	return &service{repo: repo}
}

type service struct {
	repo        repository.PaymentRepository
	creditSvc   credit.Service
	snGenerator *sequencenumber.Generator
}

// CreatePayment 创建支付记录(支付主记录 + 支付渠道流水记录) 订单模块会同步调用该模块
func (s *service) CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	// 3. 同步调用“支付模块”获取支付ID和支付SN和二维码
	//    1)创建支付, 支付记录, 冗余订单ID和订单SN
	//    2)调用“积分模块” 扣减积分
	//    3)调用“微信”, 获取二维码

	// step 0: 创建支付主记录
	p, err := s.createPayment(ctx, payment)
	if err != nil {
		return domain.Payment{}, err
	}

	// 把传递过来支付渠道相关的内容, 看作一种建议策略
	// 1. 仅积分支付, 策略失败 - fallback到微信支付
	// 2. 仅微信支付, 失败无fallback
	// 3. 积分+微信, 还是先积分,不行再微信/支付宝
	var paymentSN string
	totalAmount := payment.TotalAmount
	var createdPayment domain.Payment
	paymentDeadline := time.Now().Add(30 * time.Minute).UnixMilli()

	for _, record := range payment.Records {
		switch record.Channel {
		case domain.ChannelTypeCredit:

			sn, left, err2 := s.creditSvc.DirectDeductCredits(ctx, payment.TotalAmount)

			if err2 == nil {
				newPayment := payment
				// 直扣积分成功
				paidAt := time.Now().UnixMilli()

				newPayment.SN = paymentSN
				newPayment.PaidAt = paidAt
				newPayment.Deadline = paymentDeadline
				newPayment.Status = domain.PaymentStatusPaid

				newPayment.Records = []domain.PaymentRecord{
					{
						PaymentNO3rd: sn,
						Channel:      domain.ChannelTypeCredit,
						Amount:       payment.TotalAmount,
						PaidAt:       paidAt,
						Status:       domain.PaymentRecordStatusPaid,
					},
				}

				pp, err3 := s.repo.CreatePayment(ctx, newPayment)
				if err3 != nil {
					// todo: 事务问题, 积分扣减成功, 但是创建支付主记录及积分支付记录失败该怎么办?
					// 记录日志, 人工补偿?
					return domain.Payment{}, fmt.Errorf("创建支付主记录及积分支付记录失败: %w", err2)
				}
				return pp, nil
			}

			leftCredits := left
			var no string

			// 进最大努力预扣积分
			for leftCredits > 0 {
				// 预扣积分
				sn, l, err4 := s.creditSvc.PreDeductCredits(ctx, leftCredits)
				if err4 != nil {
					leftCredits = l
					continue
				}
				no = sn
				break
			}

			// 预扣失败
			if no == "" {
				if len(payment.Records) == 1 {
					// 仅有积分支付渠道
					return domain.Payment{}, fmt.Errorf("创建支付主记录及积分支付记录失败")
				}
				// 还有其他支付渠道
				continue
			}

			// 预扣成功
			// 创建支付主记录及积分支付记录, 状态均为未支付
			py := payment
			py.SN = paymentSN
			py.Deadline = paymentDeadline
			py.Records = []domain.PaymentRecord{
				{
					PaymentNO3rd: sn,
					Channel:      domain.ChannelTypeCredit,
					Amount:       leftCredits,
				},
			}
			pp, err5 := s.repo.CreatePayment(ctx, py)
			if err5 != nil {
				return domain.Payment{}, fmt.Errorf("创建支付主记录及积分支付记录失败: %w", err2)
			}

			// 减去已扣减的积分
			totalAmount -= leftCredits
			createdPayment = pp

			// 仅积分支付, 调用积分模块, 扣减积分
			// 积分扣减成功,拿到返回扣减后的事务ID
			//        填充, record => paymnet_3rd_no, amount, paidAt, status(已支付)等 创建主表记录(已支付)+积分扣减支付记录, 返回
			// 扣减失败? 自动fallback到微信?

		case domain.ChannelTypeWechat:

			//
			if totalAmount != payment.TotalAmount {
				// 之前有积分预扣操作
				return domain.Payment{}, nil
			} else {
				// 仅微信支付
				py := createdPayment
				py.SN = ""
			}

		}
	}

	return p, nil
}

func (s *service) createPayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	sn, err := s.snGenerator.Generate(payment.UserID)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("生成支付序列号失败: %w", err)
	}
	payment.SN = sn
	payment.Deadline = time.Now().Add(30 * time.Minute).UnixMilli()
	p, err := s.repo.CreatePayment(ctx, payment)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("创建支付主记录失败: %w", err)
	}
	return p, nil
}

func (s *service) GetPaymentChannels(ctx context.Context) []domain.PaymentChannel {
	return []domain.PaymentChannel{
		{Type: 1, Desc: "积分"},
		{Type: 2, Desc: "微信"},
	}
}

func (s *service) FindPaymentByID(ctx context.Context, id int64) (domain.Payment, error) {
	return domain.Payment{}, nil
}

// PayByOrderSN 通过订单序列号支付
func (s *service) PayByOrderSN(ctx context.Context, orderSN string) (domain.Payment, error) {
	return domain.Payment{}, nil
}

// 订单模块 调用 支付模块 创建支付记录
//    domain.Payment {ID, SN, buyer_id, orderID, orderSN, []paymentChanenl{{1, 积分, 2000}, {2, 微信, 7990, codeURL}},
//  Pay(ctx, domain.Payment) (domain.Payment[填充后],  err error)
//  含有微信就要调用 Prepay()
//  WechatPrepay(
// CreditPay(ctx,
// 1) 订单 -> 2) 支付 -> 3) 积分
