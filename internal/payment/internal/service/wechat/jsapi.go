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

	"github.com/ecodeclub/webook/internal/user"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/gotomicro/ego/core/elog"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
)

//go:generate mockgen -source=./jsapi.go -package=wechatmocks -destination=./mocks/jsapi.mock.go -typed JSAPIService
type JSAPIService interface {
	PrepayWithRequestPayment(ctx context.Context, req jsapi.PrepayRequest) (resp *jsapi.PrepayWithRequestPaymentResponse, result *core.APIResult, err error)
	QueryOrderByOutTradeNo(ctx context.Context, req jsapi.QueryOrderByOutTradeNoRequest) (resp *payments.Transaction, result *core.APIResult, err error)
}

type JSAPIPaymentService struct {
	svc     JSAPIService
	userSvc user.UserService
	basePaymentService
}

func NewJSAPIPaymentService(svc JSAPIService,
	userSvc user.UserService,
	appid, mchid, notifyURL string) *JSAPIPaymentService {
	return &JSAPIPaymentService{
		svc:     svc,
		userSvc: userSvc,
		basePaymentService: basePaymentService{
			l:         elog.DefaultLogger,
			name:      domain.ChannelTypeWechatJS,
			desc:      "微信小程序",
			appID:     appid,
			mchID:     mchid,
			notifyURL: notifyURL,
		},
	}
}

func (n *JSAPIPaymentService) Name() domain.ChannelType {
	return n.name
}

func (n *JSAPIPaymentService) Desc() string {
	return n.desc
}

func (n *JSAPIPaymentService) Prepay(ctx context.Context, pmt domain.Payment) (any, error) {
	r, ok := slice.Find(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeWechatJS
	})
	if !ok || r.Amount == 0 {
		return "", fmt.Errorf("缺少微信支付金额信息")
	}

	profile, err := n.userSvc.Profile(ctx, pmt.PayerID)
	if err != nil {
		return "", fmt.Errorf("查找用户的小程序 open id 失败 %w", err)
	}

	resp, _, err := n.svc.PrepayWithRequestPayment(ctx,
		jsapi.PrepayRequest{
			Appid:       core.String(n.appID),
			Mchid:       core.String(n.mchID),
			Description: core.String(pmt.OrderDescription),
			OutTradeNo:  core.String(pmt.OrderSN),
			TimeExpire:  core.Time(time.Now().Add(time.Minute * 30)),
			NotifyUrl:   core.String(n.notifyURL),
			Amount: &jsapi.Amount{
				Currency: core.String("CNY"),
				Total:    core.Int64(r.Amount),
			},
			Payer: &jsapi.Payer{Openid: core.String(profile.WechatInfo.MiniOpenId)},
		},
	)
	if err != nil {
		return "", fmt.Errorf("微信预支付失败: %w", err)
	}

	return domain.WechatJsAPIPrepayResponse{
		PrepayId: *resp.PrepayId,
		// 应用ID
		Appid: *resp.Appid,
		// 时间戳
		TimeStamp: *resp.TimeStamp,
		// 随机字符串
		NonceStr: *resp.NonceStr,
		// 订单详情扩展字符串
		Package: *resp.Package,
		// 签名方式
		SignType: *resp.SignType,
		// 签名
		PaySign: *resp.PaySign,
	}, nil
}

// QueryOrderBySN 同步信息 定时任务调用此方法同步状态信息
func (n *JSAPIPaymentService) QueryOrderBySN(ctx context.Context, orderSN string) (domain.Payment, error) {
	txn, _, err := n.svc.QueryOrderByOutTradeNo(ctx, jsapi.QueryOrderByOutTradeNoRequest{
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
