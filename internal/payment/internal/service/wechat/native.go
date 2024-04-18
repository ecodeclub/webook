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
	"github.com/ecodeclub/webook/internal/payment/internal/event"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/gotomicro/ego/core/elog"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
)

var errUnknownTransactionState = errors.New("未知的微信事务状态")

//go:generate mockgen -source=./native.go -package=wechatmocks -destination=./mocks/wechat.mock.go -typed NativeAPIService
type NativeAPIService interface {
	Prepay(ctx context.Context, req native.PrepayRequest) (resp *native.PrepayResponse, result *core.APIResult, err error)
	QueryOrderByOutTradeNo(ctx context.Context, req native.QueryOrderByOutTradeNoRequest) (resp *payments.Transaction, result *core.APIResult, err error)
}

type NativePaymentService struct {
	svc            NativeAPIService
	repo           repository.PaymentRepository
	producer       event.PaymentEventProducer
	paymentDDLFunc func() int64
	l              *elog.Component

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

func NewNativePaymentService(svc NativeAPIService,
	repo repository.PaymentRepository,
	producer event.PaymentEventProducer,
	paymentDDLFunc func() int64,
	l *elog.Component,
	appid, mchid string) *NativePaymentService {
	return &NativePaymentService{
		svc:            svc,
		repo:           repo,
		producer:       producer,
		paymentDDLFunc: paymentDDLFunc,
		l:              l,
		appID:          appid,
		mchID:          mchid,
		// todo: 配置回调URL
		notifyURL: "http://wechat.meoying.com/pay/callback",
		nativeCallBackTypeToPaymentStatus: map[string]domain.PaymentStatus{
			"SUCCESS":  domain.PaymentStatusPaid,
			"PAYERROR": domain.PaymentStatusFailed,
			"NOTPAY":   domain.PaymentStatusUnpaid,
			"CLOSED":   domain.PaymentStatusFailed,
			"REVOKED":  domain.PaymentStatusFailed,
			"REFUND":   domain.PaymentStatusRefund,
		},
	}
}

func (n *NativePaymentService) Prepay(ctx context.Context, pmt domain.Payment) (domain.Payment, error) {

	var amount int64
	r, ok := slice.Find(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeCredit
	})
	if !ok || r.Amount == 0 {
		return domain.Payment{}, fmt.Errorf("缺少微信支付金额信息")
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
				Total:    core.Int64(amount),
			},
		},
	)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("微信预支付失败: %w", err)
	}

	pmt.PayDDL = n.paymentDDLFunc()
	pmt.Status = domain.PaymentStatusUnpaid

	pmt.Records = []domain.PaymentRecord{
		{
			Description: pmt.OrderDescription,
			Channel:     domain.ChannelTypeWechat,
			Amount:      amount,
			Status:      domain.PaymentStatusUnpaid,
		},
	}

	pp, err2 := n.repo.CreatePayment(ctx, pmt)
	if err2 != nil {
		return domain.Payment{}, fmt.Errorf("微信预支付失败: 创建支付主记录及微信渠道支付记录失败: %w", err2)
	}

	pp.Records = slice.Map(pp.Records, func(idx int, src domain.PaymentRecord) domain.PaymentRecord {
		if src.Channel == domain.ChannelTypeWechat {
			src.WechatCodeURL = *resp.CodeUrl
		}
		return src
	})
	return pp, nil
}

// SyncWechatInfo 同步信息 定时任务调用此方法同步状态信息
func (n *NativePaymentService) SyncWechatInfo(ctx context.Context, orderSN string) error {
	txn, _, err := n.svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(orderSN),
		Mchid:      core.String(n.mchID),
	})
	if err != nil {
		return err
	}
	return n.updateByTxn(ctx, txn)
}

// FindExpiredPayment 查找过期支付记录 —— 支付主记录+微信支付记录, 定时任务会调用该方法
func (n *NativePaymentService) FindExpiredPayment(ctx context.Context, offset, limit int, t time.Time) ([]domain.Payment, error) {
	return n.repo.FindExpiredPayment(ctx, offset, limit, t)
}

// HandleCallback 处理微信回调  微信回调支付模块后会d
func (n *NativePaymentService) HandleCallback(ctx context.Context, txn *payments.Transaction) error {
	return n.updateByTxn(ctx, txn)
}

func (n *NativePaymentService) updateByTxn(ctx context.Context, txn *payments.Transaction) error {
	status, ok := n.nativeCallBackTypeToPaymentStatus[*txn.TradeState]
	if !ok {
		return fmt.Errorf("%w, %s", errUnknownTransactionState, *txn.TradeState)
	}
	// 跟新支付主记录+微信渠道支付记录两条数据的状态
	paidAt := time.Now().UnixMilli()
	pmt := domain.Payment{
		OrderSN: *txn.OutTradeNo,
		Records: []domain.PaymentRecord{
			{
				PaymentNO3rd: *txn.TransactionId,
				Channel:      domain.ChannelTypeWechat,
				PaidAt:       paidAt,
				Status:       status,
			},
		},
		Status: status,
	}

	err := n.repo.UpdatePayment(ctx, pmt)
	if err != nil {
		// 这里有一个小问题，就是如果超时了的话，你都不知道更新成功了没
		return err
	}

	// 就是处于结束状态
	err1 := n.producer.Produce(ctx, event.PaymentEvent{
		OrderSN: pmt.OrderSN,
		Status:  int64(pmt.Status),
	})
	if err1 != nil {
		// 要做好监控和告警
		n.l.Error("发送支付事件失败", elog.FieldErr(err1),
			elog.String("order_sn", pmt.OrderSN))
	}
	// 虽然发送事件失败，但是数据库记录了，所以可以返回 Nil
	return nil
}
