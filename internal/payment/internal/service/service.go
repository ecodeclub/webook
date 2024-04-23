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

	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/payment/internal/service/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/service/wechat"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/gotomicro/ego/core/elog"
)

//go:generate mockgen -source=service.go -package=paymentmocks -destination=../../mocks/payment.mock.go -typed Service
type Service interface {
	CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error)
	GetPaymentChannels(ctx context.Context) []domain.PaymentChannel
	FindPaymentByID(ctx context.Context, paymentID int64) (domain.Payment, error)
	PayByOrderID(ctx context.Context, oid int64) (domain.Payment, error)
}

func NewService(wechatSvc *wechat.NativePaymentService,
	creditSvc *credit.PaymentService,
	snGenerator *sequencenumber.Generator,
	repo repository.PaymentRepository) Service {
	return &service{
		wechatSvc:   wechatSvc,
		creditSvc:   creditSvc,
		snGenerator: snGenerator,
		repo:        repo,
		l:           elog.DefaultLogger,
	}
}

type service struct {
	wechatSvc   *wechat.NativePaymentService
	creditSvc   *credit.PaymentService
	snGenerator *sequencenumber.Generator
	repo        repository.PaymentRepository
	l           *elog.Component
}

// 订单模块 调用 支付模块 创建支付记录
//    domain.Payment {ID, SN, buyer_id, orderID, orderSN, []paymentChanenl{{1, 积分, 2000}, {2, 微信, 7990, codeURL}},
//  Pay(ctx, domain.Payment) (domain.Payment[填充后],  err error)
//  含有微信就要调用 Prepay()
//  WechatPrepay(
// CreditPay(ctx,
// 1) 订单 -> 2) 支付 -> 3) 积分

// CreatePayment 创建支付记录(支付主记录 + 支付渠道流水记录) 订单模块会同步调用该模块, 生成支付计划
func (s *service) CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {

	// 3. 同步调用“支付模块”获取支付ID和支付SN和二维码
	//    1)创建支付, 支付记录, 冗余订单ID和订单SN
	//    2)调用“积分模块” 扣减积分
	//    3)调用“微信”, 获取二维码

	// 填充公共字段
	paymentSN, err := s.snGenerator.Generate(payment.PayerID)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("生成支付序列号失败: %w", err)
	}
	payment.SN = paymentSN

	// 积分支付优先
	slices.SortFunc(payment.Records, func(a, b domain.PaymentRecord) int {
		if a.Channel < b.Channel {
			return -1
		} else if a.Channel > b.Channel {
			return 1
		}
		return 0
	})

	if len(payment.Records) == 1 {
		switch payment.Records[0].Channel {
		case domain.ChannelTypeCredit:
			// 仅积分支付
			return s.creditSvc.Pay(ctx, payment)
		case domain.ChannelTypeWechat:
			// 仅微信支付
			return s.wechatSvc.Prepay(ctx, payment)
		}
	}

	return s.prepayByWechatAndCredit(ctx, payment)
}

// prepayByWechatAndCredit 用微信和积分预支付
func (s *service) prepayByWechatAndCredit(ctx context.Context, payment domain.Payment) (domain.Payment, error) {

	p, err := s.creditSvc.Prepay(ctx, payment)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("积分与微信混合支付失败: %w", err)
	}

	pp, err2 := s.wechatSvc.Prepay(ctx, p)
	if err2 != nil {
		return domain.Payment{}, fmt.Errorf("积分与微信混合支付失败: %w", err2)
	}

	return pp, nil
}

func (s *service) GetPaymentChannels(ctx context.Context) []domain.PaymentChannel {
	return []domain.PaymentChannel{
		{Type: domain.ChannelTypeCredit, Desc: "积分"},
		{Type: domain.ChannelTypeWechat, Desc: "微信"},
	}
}

func (s *service) FindPaymentByID(ctx context.Context, id int64) (domain.Payment, error) {
	return domain.Payment{}, nil
}

// PayByOrderID 通过订单序ID支付,查找并执行支付计划
func (s *service) PayByOrderID(ctx context.Context, oid int64) (domain.Payment, error) {
	// 幂等
	return domain.Payment{}, nil
}
