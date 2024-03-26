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

	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
)

type Service interface {
	CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error)
	GetPaymentChannels(ctx context.Context) []domain.PaymentChannel
	FindPaymentByID(ctx context.Context, paymentID int64) (domain.Payment, error)
}

func NewService(repo repository.PaymentRepository) Service {
	return &service{repo: repo}
}

type service struct {
	repo repository.PaymentRepository
}

func (s *service) CreatePayment(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
	// 3. 同步调用“支付模块”获取支付ID和支付SN和二维码
	//    1)创建支付, 支付记录, 冗余订单ID和订单SN
	//    2)调用“积分模块” 扣减积分
	//    3)调用“微信”, 获取二维码
	return domain.Payment{}, nil
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

// 订单模块 调用 支付模块 创建支付记录
//    domain.Payment {ID, SN, buyer_id, orderID, orderSN, []paymentChanenl{{1, 积分, 2000}, {2, 微信, 7990, codeURL}},
//  Pay(ctx, domain.Payment) (domain.Payment[填充后],  err error)
//  含有微信就要调用 Prepay()
//  WechatPrepay(
// CreditPay(ctx,
// 1) 订单 -> 2) 支付 -> 3) 积分
