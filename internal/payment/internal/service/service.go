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
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -source=service.go -package=paymentmocks -destination=../../mocks/payment.mock.go -typed Service
type Service interface {
	CreatePayment(ctx context.Context, pmt domain.Payment) (domain.Payment, error)
	GetPaymentChannels(ctx context.Context) []domain.PaymentChannel
	FindPaymentByID(ctx context.Context, pmtID int64) (domain.Payment, error)
	PayByID(ctx context.Context, pmtID int64) (domain.Payment, error)
	// HandleWechatCallback 处理微信回调请求 web调用
	HandleWechatCallback(ctx context.Context, txn *payments.Transaction) error
	// FindTimeoutPayments 查找过期支付记录 —— 支付主记录+微信支付记录, job调用
	FindTimeoutPayments(ctx context.Context, offset int, limit int, ctime int64) ([]domain.Payment, int64, error)
	// CloseTimeoutPayment 通过支付ID关闭超时支付, job调用
	CloseTimeoutPayment(ctx context.Context, pmt domain.Payment) error
	// SyncWechatInfo 同步与微信对账 job调用
	SyncWechatInfo(ctx context.Context, pmt domain.Payment) error
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
	return pmt, err
}

func (s *service) executePayment(ctx context.Context, pmt *domain.Payment) error {
	if len(pmt.Records) == 1 {
		switch pmt.Records[0].Channel {
		case domain.ChannelTypeCredit:
			// 仅积分支付
			return s.payByCredit(ctx, pmt)
		case domain.ChannelTypeWechat:
			// 仅微信预支付
			return s.prepayByWechat(ctx, pmt)
		}
	}
	// 混合预支付
	return s.prepayByWechatAndCredit(ctx, pmt)
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
		return fmt.Errorf("预扣积分失败: %w", err)
	}

	err = s.creditSvc.ConfirmDeductCredits(ctx, pmt.PayerID, tid)
	if err != nil {
		_ = s.creditSvc.CancelDeductCredits(ctx, pmt.PayerID, tid)
		return fmt.Errorf("确认扣减积分失败: %w", err)
	}

	// 更新字段
	pmt.Status = domain.PaymentStatusPaidSuccess
	pmt.PaidAt = time.Now().UnixMilli()
	pmt.Records[idx].Status = pmt.Status
	pmt.Records[idx].PaidAt = pmt.PaidAt
	pmt.Records[idx].PaymentNO3rd = strconv.FormatInt(tid, 10)

	err = s.repo.UpdatePayment(ctx, *pmt)
	if err != nil {
		// 这里有一个小问题，就是如果超时了的话，你都不知道更新成功了没
		_ = s.creditSvc.CancelDeductCredits(ctx, pmt.PayerID, tid)
		return err
	}
	return s.sendPaymentEvent(ctx, pmt)
}

func (s *service) sendPaymentEvent(ctx context.Context, pmt *domain.Payment) error {
	// 就是处于结束状态
	evt := event.PaymentEvent{
		OrderSN: pmt.OrderSN,
		PayerID: pmt.PayerID,
		Status:  pmt.Status.ToUint8(),
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

func (s *service) prepayByWechat(ctx context.Context, pmt *domain.Payment) error {
	codeURL, err := s.wechatSvc.Prepay(ctx, *pmt)
	if err != nil {
		return err
	}
	// 设置字段
	idx := slice.IndexFunc(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeWechat
	})
	pmt.Records[idx].WechatCodeURL = codeURL
	pmt.Status = domain.PaymentStatusProcessing
	pmt.Records[idx].Status = domain.PaymentStatusProcessing
	err = s.repo.UpdatePayment(ctx, *pmt)
	if err != nil {
		// 这里有一个小问题，就是如果超时了的话，你都不知道更新成功了没
		return err
	}
	return nil
}

func (s *service) prepayByWechatAndCredit(ctx context.Context, pmt *domain.Payment) error {

	creditIdx := slice.IndexFunc(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeCredit
	})

	if creditIdx == -1 || pmt.Records[creditIdx].Amount == 0 {
		return fmt.Errorf("缺少积分支付金额信息")
	}

	tid, err := s.creditSvc.TryDeductCredits(context.Background(), credit.Credit{
		Uid: pmt.PayerID,
		Logs: []credit.CreditLog{
			{
				Key:          pmt.OrderSN,
				ChangeAmount: pmt.Records[creditIdx].Amount,
				Biz:          "order",
				BizId:        pmt.OrderID,
				Desc:         pmt.Records[creditIdx].Description,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("预扣积分失败: %w", err)
	}

	codeURL, err := s.wechatSvc.Prepay(ctx, *pmt)
	if err != nil {
		_ = s.creditSvc.CancelDeductCredits(ctx, pmt.PayerID, tid)
		return err
	}

	// 设置字段
	wechatIdx := slice.IndexFunc(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeWechat
	})

	pmt.Status = domain.PaymentStatusProcessing
	pmt.Records[wechatIdx].WechatCodeURL = codeURL
	pmt.Records[wechatIdx].Status = domain.PaymentStatusProcessing
	pmt.Records[creditIdx].PaymentNO3rd = strconv.FormatInt(tid, 10)
	pmt.Records[creditIdx].Status = domain.PaymentStatusProcessing

	err = s.repo.UpdatePayment(ctx, *pmt)
	if err != nil {
		// 这里有一个小问题，就是如果超时了的话，你都不知道更新成功了没
		_ = s.creditSvc.CancelDeductCredits(ctx, pmt.PayerID, tid)
		return err
	}
	return nil
}

func (s *service) HandleWechatCallback(ctx context.Context, txn *payments.Transaction) error {
	pmt, err := s.wechatSvc.ConvertCallbackTransactionToDomain(txn)
	if err != nil {
		return err
	}

	err = s.repo.UpdatePayment(ctx, pmt)
	if err != nil {
		// 这里有一个小问题，就是如果超时了的话，你都不知道更新成功了没
		return err
	}

	p, _ := s.repo.FindPaymentByOrderSN(context.Background(), pmt.OrderSN)

	// 支付主记录和微信支付渠道支付成功/支付失败后,就发送消息
	pmt.PayerID = p.PayerID
	_ = s.sendPaymentEvent(ctx, &pmt)

	pmt.Records = p.Records
	return s.handleCreditCallback(ctx, pmt)
}

func (s *service) handleCreditCallback(ctx context.Context, pmt domain.Payment) error {
	r, ok := slice.Find(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeCredit
	})
	if !ok {
		// 仅微信支付,直接返回nil
		return nil
	}

	uid := pmt.PayerID
	tid, _ := strconv.ParseInt(r.PaymentNO3rd, 10, 64)
	if pmt.Status == domain.PaymentStatusPaidSuccess {
		err := s.creditSvc.ConfirmDeductCredits(ctx, uid, tid)
		if err != nil {
			s.l.Warn("确认扣减积分失败",
				elog.Int64("uid", uid),
				elog.Int64("tid", tid),
			)
		}
	} else {
		s.cancelDeductCredits(ctx, uid, tid)
	}

	pp := domain.Payment{
		OrderSN: pmt.OrderSN,
		PaidAt:  pmt.PaidAt,
		Status:  pmt.Status,
		Records: []domain.PaymentRecord{
			{
				PaymentNO3rd: r.PaymentNO3rd,
				Channel:      r.Channel,
				PaidAt:       pmt.PaidAt,
				Status:       pmt.Status,
			},
		}}
	return s.repo.UpdatePayment(ctx, pp)
}

func (s *service) cancelDeductCredits(ctx context.Context, uid, tid int64) {
	err := s.creditSvc.CancelDeductCredits(ctx, uid, tid)
	if err != nil {
		s.l.Warn("确认扣减积分失败",
			elog.Int64("uid", uid),
			elog.Int64("tid", tid),
		)
	}
}

func (s *service) FindTimeoutPayments(ctx context.Context, offset int, limit int, ctime int64) ([]domain.Payment, int64, error) {

	var (
		eg    errgroup.Group
		ps    []domain.Payment
		total int64
	)
	eg.Go(func() error {
		var err error
		ps, err = s.repo.FindTimeoutPayments(ctx, offset, limit, ctime)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.TotalTimeoutPayments(ctx, ctime)
		return err
	})

	return ps, total, eg.Wait()
}

func (s *service) CloseTimeoutPayment(ctx context.Context, pmt domain.Payment) error {
	// pmt.OrderSN不能为空
	// pmt.Records不能为空
	// pmt.Status <= domain.PaymentStatusProcessing
	pmt.Status = domain.PaymentStatusTimeoutClosed
	for i := 0; i < len(pmt.Records); i++ {
		record := &pmt.Records[i]
		if record.Status == domain.PaymentStatusProcessing &&
			record.Channel == domain.ChannelTypeCredit {
			uid := pmt.PayerID
			tid, _ := strconv.ParseInt(record.PaymentNO3rd, 10, 64)
			s.cancelDeductCredits(ctx, uid, tid)
		}
		record.Status = domain.PaymentStatusTimeoutClosed
	}
	return s.repo.UpdatePayment(ctx, pmt)
}

func (s *service) SyncWechatInfo(ctx context.Context, pmt domain.Payment) error {
	p, err := s.wechatSvc.QueryOrderBySN(ctx, pmt.OrderSN)
	if err != nil {
		return err
	}

	if p.Status == domain.PaymentStatusTimeoutClosed {
		idx := slice.IndexFunc(pmt.Records, func(src domain.PaymentRecord) bool {
			return src.Channel == domain.ChannelTypeWechat
		})
		pmt.Records[idx].PaymentNO3rd = p.Records[0].PaymentNO3rd
		return s.CloseTimeoutPayment(ctx, pmt)
	}

	err = s.repo.UpdatePayment(ctx, p)
	if err != nil {
		// 这里有一个小问题，就是如果超时了的话，你都不知道更新成功了没
		return err
	}

	// 支付主记录和微信支付渠道支付成功/支付失败后,就发送消息
	p.PayerID = pmt.PayerID
	_ = s.sendPaymentEvent(ctx, &p)

	p.Records = pmt.Records
	return s.handleCreditCallback(ctx, p)
}
