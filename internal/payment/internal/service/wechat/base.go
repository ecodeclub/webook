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

package wechat

import (
	"errors"
	"fmt"
	"time"

	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/gotomicro/ego/core/elog"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
)

var (
	// TradeState2PaymentStatus 在微信 native 和 jsapi 共用回调交易信息
	TradeState2PaymentStatus = map[string]domain.PaymentStatus{
		"SUCCESS":    domain.PaymentStatusPaidSuccess, // 支付成功
		"PAYERROR":   domain.PaymentStatusPaidFailed,  // 支付失败(其他原因，如银行返回失败)
		"CLOSED":     domain.PaymentStatusPaidFailed,  // 已关闭
		"REVOKED":    domain.PaymentStatusPaidFailed,  // 已撤销（付款码支付）
		"NOTPAY":     domain.PaymentStatusUnpaid,      // 未支付
		"USERPAYING": domain.PaymentStatusProcessing,  // 用户支付中（付款码支付）
		"REFUND":     domain.PaymentStatusRefund,      // 转入退款
	}
	errUnknownTransactionState = errors.New("未知的微信交易状态")
)

type basePaymentService struct {
	l *elog.Component

	name domain.ChannelType
	desc string

	appID     string
	mchID     string
	notifyURL string
}

func (b *basePaymentService) convertToPaymentDomain(txn *payments.Transaction, status domain.PaymentStatus) domain.Payment {
	// 更新支付主记录+微信渠道支付记录两条数据的状态
	var paidAt int64
	if status == domain.PaymentStatusPaidSuccess {
		paidAt = time.Now().UnixMilli()
	}
	return domain.Payment{
		OrderSN: *txn.OutTradeNo,
		PaidAt:  paidAt,
		Status:  status,
		Records: []domain.PaymentRecord{
			{
				PaymentNO3rd: *txn.TransactionId,
				Channel:      b.name,
				PaidAt:       paidAt,
				Status:       status,
			},
		},
	}
}

func GetPaymentStatus(tradeState string) (domain.PaymentStatus, error) {
	status, ok := TradeState2PaymentStatus[tradeState]
	if !ok {
		return 0, fmt.Errorf("%w, %s", errUnknownTransactionState, tradeState)
	}
	return status, nil
}
