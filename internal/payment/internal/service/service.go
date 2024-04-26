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
	"strconv"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/event"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/payment/internal/service/wechat"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/gotomicro/ego/core/elog"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
)

//go:generate mockgen -source=service.go -package=paymentmocks -destination=../../mocks/payment.mock.go -typed Service
type Service interface {
	CreatePayment(ctx context.Context, pmt domain.Payment) (domain.Payment, error)
	GetPaymentChannels(ctx context.Context) []domain.PaymentChannel
	FindPaymentByID(ctx context.Context, pmtID int64) (domain.Payment, error)
	PayByID(ctx context.Context, pmtID int64) (domain.Payment, error)

	// HandleWechatCallback 处理微信回调请求 web调用
	HandleWechatCallback(ctx context.Context, txn *payments.Transaction) error
	// FindExpiredPayment 查找过期支付记录 —— 支付主记录+微信支付记录, job调用
	FindExpiredPayment(ctx context.Context, offset, limit int, t time.Time) ([]domain.Payment, error)
	// SyncWechatInfo 同步与微信对账 job调用
	SyncWechatInfo(ctx context.Context, orderSN string) error
}

func NewService(wechatSvc *wechat.NativePaymentService,
	creditSvc credit.Service,
	snGenerator *sequencenumber.Generator,
	repo repository.PaymentRepository,
	producer event.PaymentEventProducer,
) Service {
	return &service{
		wechatSvc:   wechatSvc,
		creditSvc:   creditSvc,
		snGenerator: snGenerator,
		repo:        repo,
		producer:    producer,
		l:           elog.DefaultLogger,
	}
}

type service struct {
	wechatSvc   *wechat.NativePaymentService
	creditSvc   credit.Service
	snGenerator *sequencenumber.Generator
	repo        repository.PaymentRepository
	producer    event.PaymentEventProducer
	l           *elog.Component
}

// CreatePayment 创建支付记录 内部不做校验
// 订单ID、订单SN、订单描述、支付金额、支付者ID、支付渠道记录不能为零值
func (s *service) CreatePayment(ctx context.Context, pmt domain.Payment) (domain.Payment, error) {
	sn, err := s.snGenerator.Generate(pmt.PayerID)
	if err != nil {
		return domain.Payment{}, err
	}
	pmt.SN = sn
	return s.repo.CreatePayment(ctx, pmt)
}

// GetPaymentChannels 获取支持的支付渠道
func (s *service) GetPaymentChannels(_ context.Context) []domain.PaymentChannel {
	return []domain.PaymentChannel{
		{Type: domain.ChannelTypeCredit, Desc: "积分"},
		{Type: domain.ChannelTypeWechat, Desc: "微信"},
	}
}

// FindPaymentByID 根据支付主记录ID查找支付记录
func (s *service) FindPaymentByID(ctx context.Context, pmtID int64) (domain.Payment, error) {
	return s.repo.FindPaymentByID(ctx, pmtID)
}

// PayByID 通过支付主记录ID支付,查找并执行支付计划
func (s *service) PayByID(ctx context.Context, pmtID int64) (domain.Payment, error) {
	pmt, err := s.FindPaymentByID(ctx, pmtID)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("查找支付记录失败: %w, pmtID: %d", err, pmtID)
	}

	err = s.executePayment(ctx, &pmt)
	if err != nil {
		return domain.Payment{}, fmt.Errorf("执行支付操作失败: %w, pmtID: %d", err, pmtID)
	}

	// 幂等 判定状态,如果为processing, 忽略更新
	return pmt, err
}

func (s *service) executePayment(ctx context.Context, pmt *domain.Payment) error {
	// 积分支付优先
	slices.SortFunc(pmt.Records, func(a, b domain.PaymentRecord) int {
		if a.Channel < b.Channel {
			return -1
		} else if a.Channel > b.Channel {
			return 1
		}
		return 0
	})

	var err error
	if len(pmt.Records) == 1 {
		switch pmt.Records[0].Channel {
		case domain.ChannelTypeCredit:
			// 仅积分支付
			err = s.payByCredit(ctx, pmt)
		case domain.ChannelTypeWechat:
			// 仅微信支付
			return nil
		}
	}

	if err != nil {
		return err
	}

	return s.sendPaymentEvent(ctx, pmt)
}

func (s *service) payByCredit(ctx context.Context, pmt *domain.Payment) error {

	idx := slice.IndexFunc(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeCredit
	})

	if idx == -1 || pmt.Records[idx].Amount == 0 {
		return fmt.Errorf("缺少积分支付金额信息")
	}

	tid, err := s.creditSvc.TryDeductCredits(ctx, credit.Credit{
		Uid: pmt.PayerID,
		Logs: []credit.CreditLog{
			{
				Key:          pmt.OrderSN,
				ChangeAmount: pmt.Records[idx].Amount,
				Biz:          "order",
				BizId:        pmt.OrderID,
				Desc:         pmt.Records[idx].Description,
			},
		},
	})
	if err != nil {
		return err
	}

	err = s.creditSvc.ConfirmDeductCredits(ctx, pmt.PayerID, tid)
	if err != nil {
		return err
	}

	// 更新字段
	s.setPaymentFields(pmt, idx, strconv.FormatInt(tid, 10))
	err = s.repo.UpdatePayment(ctx, *pmt)
	if err != nil {
		// 这里有一个小问题，就是如果超时了的话，你都不知道更新成功了没
		return err
	}
	return nil
}

func (s *service) setPaymentFields(pmt *domain.Payment, idx int, paymentNO3rd string) {
	pmt.Status = domain.PaymentStatusPaidSuccess
	pmt.PaidAt = time.Now().UnixMilli()
	pmt.Records[idx].Status = pmt.Status
	pmt.Records[idx].PaidAt = pmt.PaidAt
	pmt.Records[idx].PaymentNO3rd = paymentNO3rd
}

func (s *service) sendPaymentEvent(ctx context.Context, pmt *domain.Payment) error {
	// 就是处于结束状态
	evt := event.PaymentEvent{
		OrderSN: pmt.OrderSN,
		PayerID: pmt.PayerID,
		Status:  uint8(pmt.Status),
	}
	err := s.producer.Produce(ctx, evt)
	if err != nil {
		// 要做好监控和告警
		s.l.Error("发送支付事件失败",
			elog.FieldErr(err),
			elog.Any("evt", evt))
	}
	// 虽然发送事件失败，但是数据库记录了，所以可以返回 Nil
	return nil
}

// 订单模块 调用 支付模块 创建支付记录
//    domain.Payment {ID, SN, buyer_id, orderID, orderSN, []paymentChanenl{{1, 积分, 2000}, {2, 微信, 7990, codeURL}},
//  Pay(ctx, domain.Payment) (domain.Payment[填充后],  err error)
//  含有微信就要调用 Prepay()
//  WechatPrepay(
// CreditPay(ctx,
// 1) 订单 -> 2) 支付 -> 3) 积分

// CreatePaymentV1 创建支付记录(支付主记录 + 支付渠道流水记录) 订单模块会同步调用该模块, 生成支付计划
func (s *service) CreatePaymentV1(ctx context.Context, payment domain.Payment) (domain.Payment, error) {
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
			// return s.creditSvc.Pay(ctx, payment)
			return domain.Payment{}, nil
		case domain.ChannelTypeWechat:
			// 仅微信支付
			// return s.wechatSvc.Prepay(ctx, payment)
			return domain.Payment{}, nil
		}
	}

	return s.prepayByWechatAndCredit(ctx, payment)
}

// prepayByWechatAndCredit 用微信和积分预支付
func (s *service) prepayByWechatAndCredit(ctx context.Context, pmt domain.Payment) (domain.Payment, error) {

	// p, err := s.creditSvc.Prepay(ctx, payment)
	// if err != nil {
	// 	return domain.Payment{}, fmt.Errorf("积分与微信混合支付失败: %w", err)
	// }
	// pp, err2 := s.wechatSvc.Prepay(ctx, pmt)
	// if err2 != nil {
	// 	return domain.Payment{}, fmt.Errorf("积分与微信混合支付失败: %w", err2)
	// }
	//
	// return pp, nil
	return domain.Payment{}, nil
}

func (s *service) HandleWechatCallback(ctx context.Context, txn *payments.Transaction) error {
	pmt, err := s.wechatSvc.ConvertTransactionToDomain(txn)
	if err != nil {
		return err
	}

	err = s.repo.UpdatePayment(ctx, pmt)
	if err != nil {
		// 这里有一个小问题，就是如果超时了的话，你都不知道更新成功了没
		return err
	}

	err = s.sendPaymentEvent(ctx, &pmt)
	if err != nil {
		return err
	}

	// 如果为混合支付,执行积分回调操作
	return s.handleCreditCallback(ctx, pmt)
}

func (s *service) handleCreditCallback(ctx context.Context, pmt domain.Payment) error {
	// 查找支付主记录及是否含有积分支付渠道记录
	// 如果找到, 通过pmt,生成新的主记录+积分渠道记录
	// s.repo.UpdatePayment()
	return nil
}

func (s *service) FindExpiredPayment(ctx context.Context, offset, limit int, t time.Time) ([]domain.Payment, error) {
	return s.repo.FindExpiredPayment(ctx, offset, limit, t)
}

func (s *service) SyncWechatInfo(ctx context.Context, orderSN string) error {
	txn, err := s.wechatSvc.QueryOrderBySN(ctx, orderSN)
	if err != nil {
		return err
	}
	return s.HandleWechatCallback(ctx, txn)
}
