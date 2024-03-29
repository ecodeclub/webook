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
	"slices"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/payment/internal/service/wechat"
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
	wechatSvc   *wechat.NativePaymentService
	creditSvc   credit.Service
	snGenerator *sequencenumber.Generator
	repo        repository.PaymentRepository
}

// CreatePayment 创建支付记录(支付主记录 + 支付渠道流水记录) 订单模块会同步调用该模块
func (s *service) CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	// 3. 同步调用“支付模块”获取支付ID和支付SN和二维码
	//    1)创建支付, 支付记录, 冗余订单ID和订单SN
	//    2)调用“积分模块” 扣减积分
	//    3)调用“微信”, 获取二维码

	// 填充公共字段
	paymentSN, err := s.snGenerator.Generate(payment.UserID)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("生成支付序列号失败: %w", err)
	}
	payment.SN = paymentSN
	payment.Deadline = time.Now().Add(30 * time.Minute).UnixMilli()

	// 积分支付优先
	slices.SortFunc(payment.Records, func(a, b domain.PaymentRecord) int {
		if a.Channel < b.Channel {
			return -1
		} else if a.Channel > b.Channel {
			return 1
		}
		return 0
	})

	// 把传递过来支付渠道相关的内容, 看作一种建议策略
	// 1. 仅积分支付, 策略失败 - fallback到微信支付
	// 2. 仅微信支付, 失败无fallback
	// 3. 积分+微信, 还是先积分,不行再微信/支付宝
	totalAmount := payment.TotalAmount
	var createdPayment domain.Payment

	for _, record := range payment.Records {
		switch record.Channel {
		case domain.ChannelTypeCredit:

			// 直接扣减, 扣减失败会返回可用积分
			paymentNO3rd, left, err := s.creditSvc.DirectDeductCredits(ctx, payment.TotalAmount)
			if err == nil {
				// 直扣积分成功
				return s.createPaidPaymentAndCreditPaymentRecord(ctx, payment, paymentNO3rd)
			}

			// 进最大努力预扣积分
			leftCredits := left
			for leftCredits > 0 {
				// 预扣积分
				no3rd, l, err2 := s.creditSvc.PreDeductCredits(ctx, leftCredits)
				if err2 != nil {
					// 预扣失败, 更新可用积分
					leftCredits = l
					continue
				}
				paymentNO3rd = no3rd
				break
			}

			// 预扣失败
			if paymentNO3rd == "" {
				if len(payment.Records) == 1 {
					// 仅有积分支付渠道
					return domain.Payment{}, fmt.Errorf("创建支付主记录及积分支付记录失败")
				}
				// 还有其他支付渠道
				continue
			}

			// 预扣成功
			prePaidAmount := leftCredits
			p, err3 := s.createUnpaidPayment(ctx, payment, domain.PaymentRecord{
				PaymentNO3rd: paymentNO3rd,
				Channel:      domain.ChannelTypeCredit,
				Amount:       prePaidAmount,
			})
			if err3 != nil {
				return p, err3
			}

			// 减去已扣减的积分
			totalAmount -= prePaidAmount
			createdPayment = p

			// 仅积分支付, 调用积分模块, 扣减积分
			// 积分扣减成功,拿到返回扣减后的事务ID
			//        填充, record => paymnet_3rd_no, amount, paidAt, status(已支付)等 创建主表记录(已支付)+积分扣减支付记录, 返回
			// todo: 扣减失败? 自动fallback到微信?

		case domain.ChannelTypeWechat:

			// 触发微信支付流程, 获取支付二维码
			codeURL, err4 := s.wechatSvc.Prepay(ctx, payment)
			if err4 != nil {
				return domain.Payment{}, err4
			}

			// todo: 如何拿到微信的txn_id来填充 paymentNO3rd
			var paymentNO3rd string

			// 之前预扣积分已经创建积分支付记录,
			if totalAmount != payment.TotalAmount {
				// 仅创建微信支付记录即可
				_, err5 := s.repo.CreatePaymentRecord(ctx, domain.PaymentRecord{
					PaymentID:    createdPayment.ID,
					PaymentNO3rd: paymentNO3rd,
					Channel:      domain.ChannelTypeWechat,
					Amount:       totalAmount,
					PaidAt:       time.Now().UnixMilli(),
					Status:       domain.PaymentStatusUnpaid,
				})
				if err5 != nil {
					return domain.Payment{}, fmt.Errorf("创建微信支付记录失败: %w", err5)
				}
				// 返回包含主记录+积分支付记录+微信支付记录
				p, err7 := s.FindPaymentByID(ctx, createdPayment.ID)
				if err7 != nil {
					return domain.Payment{}, fmt.Errorf("获取: %w", err7)
				}

				// 填充URL
				p.Records = slice.Map(p.Records, func(idx int, src domain.PaymentRecord) domain.PaymentRecord {
					if src.Channel == domain.ChannelTypeWechat {
						src.WechatCodeURL = codeURL
					}
					return src
				})
				return p, nil
			}

			// 仅微信支付, 创建支付主记录和微信支付记录
			pp, err6 := s.createUnpaidPayment(ctx, payment, domain.PaymentRecord{
				PaymentNO3rd: paymentNO3rd,
				Channel:      domain.ChannelTypeWechat,
				Amount:       totalAmount,
			})
			if err6 != nil {
				return domain.Payment{}, err6
			}

			// 填充二维码
			pp.Records = slice.Map(pp.Records, func(idx int, src domain.PaymentRecord) domain.PaymentRecord {
				if src.Channel == domain.ChannelTypeWechat {
					src.WechatCodeURL = codeURL
				}
				return src
			})
			return pp, nil
		}
	}

	return createdPayment, nil
}

// createPaidPaymentAndCreditPaymentRecord 创建已支付支付主记录及积分支付记录
func (s *service) createPaidPaymentAndCreditPaymentRecord(ctx context.Context, payment domain.Payment, paymentNO3rd string) (domain.Payment, error) {

	paidAt := time.Now().UnixMilli()
	payment.PaidAt = paidAt
	payment.Status = domain.PaymentStatusPaid

	payment.Records = []domain.PaymentRecord{
		{
			PaymentNO3rd: paymentNO3rd,
			Channel:      domain.ChannelTypeCredit,
			Amount:       payment.TotalAmount,
			PaidAt:       paidAt,
			Status:       domain.PaymentRecordStatusPaid,
		},
	}

	pp, err := s.repo.CreatePayment(ctx, payment)
	if err != nil {
		// todo: 事务问题, 积分扣减成功, 但是创建支付主记录及积分支付记录失败该怎么办?
		// 记录日志, 人工补偿?
		return domain.Payment{}, fmt.Errorf("创建支付主记录及积分支付记录失败: %w", err)
	}
	return pp, nil
}

func (s *service) createUnpaidPayment(ctx context.Context, payment domain.Payment, record domain.PaymentRecord) (domain.Payment, error) {
	payment.Records = []domain.PaymentRecord{record}
	pp, err2 := s.repo.CreatePayment(ctx, payment)
	if err2 != nil {
		return domain.Payment{}, fmt.Errorf("创建支付主记录及积分支付记录失败: %w", err2)
	}
	return pp, nil
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
