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
	"fmt"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/gotomicro/ego/core/elog"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

//go:generate mockgen -source=./native.go -package=wechatmocks -destination=./mocks/native.mock.go -typed NativeAPIService
type NativeAPIService interface {
	Prepay(ctx context.Context, req native.PrepayRequest) (resp *native.PrepayResponse, result *core.APIResult, err error)
	QueryOrderByOutTradeNo(ctx context.Context, req native.QueryOrderByOutTradeNoRequest) (resp *payments.Transaction, result *core.APIResult, err error)
}

type NativePaymentService struct {
	svc NativeAPIService
	basePaymentService
}

func NewNativePaymentService(svc NativeAPIService, appid, mchid, notifyURL string) *NativePaymentService {
	return &NativePaymentService{
		svc: svc,
		basePaymentService: basePaymentService{
			l:         elog.DefaultLogger,
			name:      domain.ChannelTypeWechat,
			desc:      "微信",
			appID:     appid,
			mchID:     mchid,
			notifyURL: notifyURL,
		},
	}
}

func (n *NativePaymentService) Name() domain.ChannelType {
	return n.name
}

func (n *NativePaymentService) Desc() string {
	return n.desc
}

func (n *NativePaymentService) Prepay(ctx context.Context, pmt domain.Payment) (any, error) {

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

// QueryOrderBySN 同步信息 定时任务调用此方法同步状态信息
func (n *NativePaymentService) QueryOrderBySN(ctx context.Context, orderSN string) (domain.Payment, error) {
	txn, _, err := n.svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(orderSN),
		Mchid:      core.String(n.mchID),
	})
	if err != nil {
		return domain.Payment{}, err
	}

	status, err := GetPaymentStatus(*txn.TradeState)
	if err != nil {
		return domain.Payment{}, err
	}

	if status != domain.PaymentStatusPaidSuccess && status != domain.PaymentStatusPaidFailed {
		// 主动同步时不再忽略,而是直接标记为超时
		status = domain.PaymentStatusTimeoutClosed
	}
	return n.convertToPaymentDomain(txn, status), nil
}
