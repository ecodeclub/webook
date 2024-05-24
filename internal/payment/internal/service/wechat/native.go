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
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/gotomicro/ego/core/elog"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

var (
	errUnknownTransactionState = errors.New("未知的微信事务状态")
	errIgnoredPaymentStatus    = errors.New("忽略的支付状态")
)

//go:generate mockgen -source=./native.go -package=wechatmocks -destination=./mocks/native.mock.go -typed NativeAPIService
type NativeAPIService interface {
	Prepay(ctx context.Context, req native.PrepayRequest) (resp *native.PrepayResponse, result *core.APIResult, err error)
	QueryOrderByOutTradeNo(ctx context.Context, req native.QueryOrderByOutTradeNoRequest) (resp *payments.Transaction, result *core.APIResult, err error)
}

type NativePaymentService struct {
	svc NativeAPIService
	l   *elog.Component

	appID     string
	mchID     string
	notifyURL string
	// 在微信 native 里面，分别是
	// SUCCESS：支付成功
	// REFUND：转入退款
	// NOTPAY：未支付
	// CLOSED：已关闭
	// REVOKED：已撤销（付款码支付）
	// USERPAYING：用户支付中（付款码支付）
	// PAYERROR：支付失败(其他原因，如银行返回失败)
	nativeCallBackTypeToPaymentStatus map[string]domain.PaymentStatus
}

func NewNativePaymentService(svc NativeAPIService, appid, mchid string) *NativePaymentService {
	return &NativePaymentService{
		svc:       svc,
		l:         elog.DefaultLogger,
		appID:     appid,
		mchID:     mchid,
		notifyURL: "http://wechat.meoying.com/api/interview/pay/callback",
		nativeCallBackTypeToPaymentStatus: map[string]domain.PaymentStatus{
			"SUCCESS":    domain.PaymentStatusPaidSuccess, // 支付成功
			"PAYERROR":   domain.PaymentStatusPaidFailed,  // 支付失败(其他原因，如银行返回失败)
			"CLOSED":     domain.PaymentStatusPaidFailed,  // 已关闭
			"REVOKED":    domain.PaymentStatusPaidFailed,  // 已撤销（付款码支付）
			"NOTPAY":     domain.PaymentStatusUnpaid,      // 未支付
			"USERPAYING": domain.PaymentStatusProcessing,  // 用户支付中（付款码支付）
			"REFUND":     domain.PaymentStatusRefund,      // 转入退款
		},
	}
}

func (n *NativePaymentService) Prepay(ctx context.Context, pmt domain.Payment) (string, error) {

	r, ok := slice.Find(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeWechat
	})
	if !ok || r.Amount == 0 {
		return "", fmt.Errorf("缺少微信支付金额信息")
	}

	resp, _, err := n.svc.Prepay(ctx,
		native.PrepayRequest{
			Appid:       core.String(n.appID),
			Mchid:       core.String(n.mchID),
			Description: core.String(pmt.OrderDescription),
			OutTradeNo:  core.String(pmt.OrderSN),
			TimeExpire:  core.Time(time.Now().Add(time.Minute * 30)),
			NotifyUrl:   core.String(n.notifyURL),
			Amount: &native.Amount{
				Currency: core.String("CNY"),
				Total:    core.Int64(r.Amount),
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("微信预支付失败: %w", err)
	}

	return *resp.CodeUrl, nil
}

func (n *NativePaymentService) ConvertCallbackTransactionToDomain(txn *payments.Transaction) (domain.Payment, error) {
	status, err := n.convertoPaymentStatus(*txn.TradeState)
	if err != nil {
		return domain.Payment{}, err
	}

	if status != domain.PaymentStatusPaidSuccess && status != domain.PaymentStatusPaidFailed {
		n.l.Warn("忽略的微信支付通知状态",
			elog.String("TradeState", *txn.TradeState),
			elog.Any("PaymentStatus", status),
		)
		return domain.Payment{}, fmt.Errorf("%w, %d", errIgnoredPaymentStatus, status.ToUint8())
	}

	return n.convertToPaymentDomain(txn, status), nil
}

func (n *NativePaymentService) convertoPaymentStatus(tradeState string) (domain.PaymentStatus, error) {
	status, ok := n.nativeCallBackTypeToPaymentStatus[tradeState]
	if !ok {
		return 0, fmt.Errorf("%w, %s", errUnknownTransactionState, tradeState)
	}
	return status, nil
}

func (n *NativePaymentService) convertToPaymentDomain(txn *payments.Transaction, status domain.PaymentStatus) domain.Payment {
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
				Channel:      domain.ChannelTypeWechat,
				PaidAt:       paidAt,
				Status:       status,
			},
		},
	}
}

// QueryOrderBySN 同步信息 定时任务调用此方法同步状态信息
func (n *NativePaymentService) QueryOrderBySN(ctx context.Context, orderSN string) (domain.Payment, error) {
	txn, _, err := n.svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(orderSN),
		Mchid:      core.String(n.mchID),
	})
	if err != nil {
		return domain.Payment{}, err
	}

	status, err := n.convertoPaymentStatus(*txn.TradeState)
	if err != nil {
		return domain.Payment{}, err
	}

	if status != domain.PaymentStatusPaidSuccess && status != domain.PaymentStatusPaidFailed {
		// 主动同步时不再忽略,而是直接标记为超时
		status = domain.PaymentStatusTimeoutClosed
	}
	return n.convertToPaymentDomain(txn, status), nil
}
