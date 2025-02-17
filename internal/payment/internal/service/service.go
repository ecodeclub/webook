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
	"errors"
	"fmt"
	"slices"
	"sort"
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

var (
	errInvalidCombinationPayment = errors.New("非法组合支付")
	errIgnoredPaymentStatus      = errors.New("忽略的支付状态")
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

	// HandleCreditCallback 处理积分支付的回调 recon模块使用
	HandleCreditCallback(ctx context.Context, pmt domain.Payment) error
	// SetPaymentStatusPaidFailed 将支付标记为失败并发送相应事件 recon模块使用
	SetPaymentStatusPaidFailed(ctx context.Context, pmt *domain.Payment) error
}

// PaymentService 封装底层不同支付方式，当前有Native支付和JSAPI支付
type PaymentService interface {
	Name() domain.ChannelType
	Desc() string
	// Prepay 预支付 Native支付方式返回CodeUrl string，JSAPI支付方式返回PrepayId
	Prepay(ctx context.Context, pmt domain.Payment) (any, error)
	// QueryOrderBySN 同步信息 定时任务调用此方法同步状态信息
	QueryOrderBySN(ctx context.Context, orderSN string) (domain.Payment, error)
}

func NewService(paymentSvcs map[domain.ChannelType]PaymentService,
	creditSvc credit.Service,
	snGenerator *sequencenumber.Generator,
	repo repository.PaymentRepository,
	producer event.PaymentEventProducer,
) Service {
	return &service{
		thirdPartyPayments: paymentSvcs,
		creditSvc:          creditSvc,
		snGenerator:        snGenerator,
		repo:               repo,
		producer:           producer,
		l:                  elog.DefaultLogger,
	}
}

type service struct {
	thirdPartyPayments map[domain.ChannelType]PaymentService
	creditSvc          credit.Service
	snGenerator        *sequencenumber.Generator
	repo               repository.PaymentRepository
	producer           event.PaymentEventProducer
	l                  *elog.Component
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
	channels := make([]domain.PaymentChannel, 0, len(s.thirdPartyPayments))
	for _, v := range s.thirdPartyPayments {
		channels = append(channels, domain.PaymentChannel{Type: v.Name(), Desc: v.Desc()})
	}
	channels = append(channels, domain.PaymentChannel{Type: domain.ChannelTypeCredit, Desc: "积分"})
	// 按 Type 升序排序
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Type < channels[j].Type
	})
	return channels
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

	channels := s.getChannelTypes(pmt)
	switch len(channels) {
	case 1:
		switch channels[0] {
		case domain.ChannelTypeCredit:
			// 仅积分支付，直接支付
			return s.payByCredit(ctx, pmt)
		case domain.ChannelTypeWechat, domain.ChannelTypeWechatJS:
			// 仅微信预支付
			return s.prepay(ctx, pmt, channels[0])
		}
	case 2:
		if channels[0] != domain.ChannelTypeCredit || channels[1] == domain.ChannelTypeCredit {
			return errInvalidCombinationPayment
		}
		// 混合预支付
		return s.prepayByCreditAnd3rdPayment(ctx, pmt, channels[1])
	}
	return errInvalidCombinationPayment
}

func (s *service) getChannelTypes(pmt *domain.Payment) []domain.ChannelType {
	channels := slice.Map(pmt.Records, func(idx int, src domain.PaymentRecord) domain.ChannelType {
		return pmt.Records[idx].Channel
	})
	slices.Sort(channels)
	return channels
}

func (s *service) payByCredit(ctx context.Context, pmt *domain.Payment) error {

	idx, tid, err := s.getCreditIndexAndDeductID(ctx, pmt)
	if err != nil {
		return err
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

func (s *service) getCreditIndexAndDeductID(ctx context.Context, pmt *domain.Payment) (int, int64, error) {
	idx := slice.IndexFunc(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeCredit
	})

	if idx == -1 || pmt.Records[idx].Amount == 0 {
		return 0, 0, fmt.Errorf("缺少积分支付金额信息")
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
		return 0, 0, fmt.Errorf("预扣积分失败: %w", err)
	}
	return idx, tid, nil
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

func (s *service) prepay(ctx context.Context, pmt *domain.Payment, channel domain.ChannelType) error {
	thirdPartyPayment := s.thirdPartyPayments[channel]
	resp, err := thirdPartyPayment.Prepay(ctx, *pmt)
	if err != nil {
		return err
	}
	// 设置字段
	idx := slice.IndexFunc(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == thirdPartyPayment.Name()
	})
	if thirdPartyPayment.Name() == domain.ChannelTypeWechat {
		pmt.Records[idx].WechatCodeURL = resp.(string)
	} else if thirdPartyPayment.Name() == domain.ChannelTypeWechatJS {
		pmt.Records[idx].WechatJsAPIResp = resp.(domain.WechatJsAPIPrepayResponse)
	}

	pmt.Status = domain.PaymentStatusProcessing
	pmt.Records[idx].Status = domain.PaymentStatusProcessing
	return s.repo.UpdatePayment(ctx, *pmt)
}

func (s *service) prepayByCreditAnd3rdPayment(ctx context.Context, pmt *domain.Payment, channel domain.ChannelType) error {

	creditIdx, tid, err := s.getCreditIndexAndDeductID(ctx, pmt)
	if err != nil {
		return err
	}

	thirdPartyPayment := s.thirdPartyPayments[channel]
	resp, err := thirdPartyPayment.Prepay(ctx, *pmt)
	if err != nil {
		_ = s.creditSvc.CancelDeductCredits(ctx, pmt.PayerID, tid)
		return err
	}

	// 设置字段
	channelIdx := slice.IndexFunc(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == thirdPartyPayment.Name()
	})

	pmt.Status = domain.PaymentStatusProcessing
	if thirdPartyPayment.Name() == domain.ChannelTypeWechat {
		pmt.Records[channelIdx].WechatCodeURL = resp.(string)
	} else if thirdPartyPayment.Name() == domain.ChannelTypeWechatJS {
		pmt.Records[channelIdx].WechatJsAPIResp = resp.(domain.WechatJsAPIPrepayResponse)
	}
	pmt.Records[channelIdx].Status = domain.PaymentStatusProcessing
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

	status, err := s.getPaymentStatus(*txn.TradeState)
	if err != nil {
		return err
	}

	pmt, err1 := s.repo.FindPaymentByOrderSN(ctx, *txn.OutTradeNo)
	if err1 != nil {
		return fmt.Errorf("微信回调中携带的订单SN不存在：%w", err1)
	}

	pmt.PaidAt = s.getPaymentPaidAt(status)
	pmt.Status = status
	for i, r := range pmt.Records {
		if r.Channel == domain.ChannelTypeWechat || r.Channel == domain.ChannelTypeWechatJS {
			pmt.Records[i] = domain.PaymentRecord{
				PaymentID:       r.PaymentID,
				PaymentNO3rd:    *txn.TransactionId,
				Description:     r.Description,
				Channel:         r.Channel,
				Amount:          r.Amount,
				PaidAt:          pmt.PaidAt,
				Status:          pmt.Status,
				WechatCodeURL:   r.WechatCodeURL,
				WechatJsAPIResp: r.WechatJsAPIResp,
			}
		}
	}

	err = s.repo.UpdatePayment(ctx, pmt)
	if err != nil {
		// 这里有一个小问题，就是如果超时了的话，你都不知道更新成功了没
		return err
	}

	// 支付主记录和微信支付渠道记录更新后就直接发送消息
	_ = s.sendPaymentEvent(ctx, &pmt)

	return s.HandleCreditCallback(ctx, pmt)
}

func (s *service) getPaymentPaidAt(status domain.PaymentStatus) int64 {
	var paidAt int64
	if status == domain.PaymentStatusPaidSuccess {
		paidAt = time.Now().UnixMilli()
	}
	return paidAt
}

func (s *service) getPaymentStatus(tradeState string) (domain.PaymentStatus, error) {
	// 将微信的交易状态转换为webook内部对应的支付状态
	status, err := wechat.GetPaymentStatus(tradeState)
	if err != nil {
		return 0, err
	}

	// 被动等待微信回调时，忽略除支付成功和支付失败之外的状态
	if status != domain.PaymentStatusPaidSuccess && status != domain.PaymentStatusPaidFailed {
		s.l.Warn("忽略的微信支付通知状态",
			elog.String("TradeState", tradeState),
			elog.Any("PaymentStatus", status),
		)
		return 0, fmt.Errorf("%w, %d", errIgnoredPaymentStatus, status.ToUint8())
	}
	return status, nil
}

func (s *service) HandleCreditCallback(ctx context.Context, pmt domain.Payment) error {
	r, ok := slice.Find(pmt.Records, func(src domain.PaymentRecord) bool {
		return src.Channel == domain.ChannelTypeCredit
	})
	if !ok {
		// 仅微信支付,直接返回nil
		return nil
	}

	if r.Status == pmt.Status &&
		r.PaidAt == pmt.PaidAt {
		return nil
	}

	if r.PaymentNO3rd != "" {
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
		record.Status = pmt.Status
	}
	return s.repo.UpdatePayment(ctx, pmt)
}

func (s *service) SyncWechatInfo(ctx context.Context, pmt domain.Payment) error {

	channelTypes := s.getChannelTypes(&pmt)
	channelType := channelTypes[len(channelTypes)-1]
	p, err := s.thirdPartyPayments[channelType].QueryOrderBySN(ctx, pmt.OrderSN)
	if err != nil {
		return err
	}

	if p.Status == domain.PaymentStatusTimeoutClosed {
		idx := slice.IndexFunc(pmt.Records, func(src domain.PaymentRecord) bool {
			return src.Channel == channelType
		})
		// p.Records是通过第三方支付响应构造的，所以Records中只有一个元素可以安全使用0
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
	return s.HandleCreditCallback(ctx, p)
}

func (s *service) SetPaymentStatusPaidFailed(ctx context.Context, pmt *domain.Payment) error {
	pmt.Status = domain.PaymentStatusPaidFailed
	for i := 0; i < len(pmt.Records); i++ {
		record := &pmt.Records[i]
		if record.Status == domain.PaymentStatusProcessing &&
			record.Channel == domain.ChannelTypeCredit {
			uid := pmt.PayerID
			tid, _ := strconv.ParseInt(record.PaymentNO3rd, 10, 64)
			s.cancelDeductCredits(ctx, uid, tid)
		}
		record.Status = pmt.Status
	}
	return s.repo.UpdatePayment(ctx, *pmt)
}
