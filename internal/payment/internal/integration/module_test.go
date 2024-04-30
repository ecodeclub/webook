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

//go:build e2e

package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/credit"
	creditmocks "github.com/ecodeclub/webook/internal/credit/mocks"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/event"
	evtmocks "github.com/ecodeclub/webook/internal/payment/internal/event/mocks"
	startup "github.com/ecodeclub/webook/internal/payment/internal/integration/setup"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
	wechatmocks "github.com/ecodeclub/webook/internal/payment/internal/service/wechat/mocks"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"go.uber.org/mock/gomock"
)

func TestPaymentModule(t *testing.T) {
	suite.Run(t, new(PaymentModuleTestSuite))
}

type PaymentModuleTestSuite struct {
	suite.Suite
	db *egorm.Component
}

func (s *PaymentModuleTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	startup.InitDAO(s.db)
}

func (s *PaymentModuleTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `payments`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `payment_records`").Error
	require.NoError(s.T(), err)
}

func (s *PaymentModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `payments`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `payment_records`").Error
	require.NoError(s.T(), err)
}

func (s *PaymentModuleTestSuite) TestService_CreatePayment() {
	t := s.T()

	testCases := []struct {
		name           string
		pmt            domain.Payment
		newSvcFunc     func(t *testing.T, ctrl *gomock.Controller) service.Service
		errRequireFunc require.ErrorAssertionFunc
		after          func(t *testing.T, svc service.Service, expected payment.Payment)
	}{
		{
			name: "仅积分支付_首次创建支付记录成功",
			pmt: domain.Payment{
				OrderID:          100001,
				OrderSN:          "create-payment-100001",
				PayerID:          100001,
				OrderDescription: "月会员 * 1",
				TotalAmount:      990,
				Records: []domain.PaymentRecord{
					{
						Description: "月会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      990,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, expected payment.Payment) {
				actual, err := svc.FindPaymentByID(context.Background(), expected.ID)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			},
		},
		{
			name: "仅积分支付_相同订单ID和SN查找支付记录成功",
			pmt: domain.Payment{
				OrderID:          100001,
				OrderSN:          "create-payment-100001",
				PayerID:          100001,
				OrderDescription: "月会员 * 1",
				TotalAmount:      990,
				Records: []domain.PaymentRecord{
					{
						Description: "月会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      990,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, expected payment.Payment) {
				actual, err := svc.FindPaymentByID(context.Background(), expected.ID)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			},
		},
		{
			name: "仅微信支付_首次创建支付记录成功",
			pmt: domain.Payment{
				OrderID:          100002,
				OrderSN:          "create-payment-100002",
				PayerID:          100002,
				OrderDescription: "季会员 * 1",
				TotalAmount:      10000,
				Records: []domain.PaymentRecord{
					{
						Description: "季会员 * 1",
						Channel:     domain.ChannelTypeWechat,
						Amount:      10000,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, expected payment.Payment) {
				actual, err := svc.FindPaymentByID(context.Background(), expected.ID)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			},
		},
		{
			name: "仅微信支付_相同订单ID和SN查找支付记录成功",
			pmt: domain.Payment{
				OrderID:          100002,
				OrderSN:          "create-payment-100002",
				PayerID:          100002,
				OrderDescription: "季会员 * 1",
				TotalAmount:      10000,
				Records: []domain.PaymentRecord{
					{
						Description: "季会员 * 1",
						Channel:     domain.ChannelTypeWechat,
						Amount:      10000,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, expected payment.Payment) {
				actual, err := svc.FindPaymentByID(context.Background(), expected.ID)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			},
		},
		{
			name: "混合支付_首次创建支付记录成功",
			pmt: domain.Payment{
				OrderID:          100003,
				OrderSN:          "create-payment-100003",
				PayerID:          100003,
				OrderDescription: "年会员 * 1",
				TotalAmount:      30000,
				Records: []domain.PaymentRecord{
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeWechat,
						Amount:      10000,
					},
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      20000,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, expected payment.Payment) {
				actual, err := svc.FindPaymentByID(context.Background(), expected.ID)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			},
		},
		{
			name: "混合支付_相同订单ID和SN查找成功",
			pmt: domain.Payment{
				OrderID:          100003,
				OrderSN:          "create-payment-100003",
				PayerID:          100003,
				OrderDescription: "年会员 * 1",
				TotalAmount:      30000,
				Records: []domain.PaymentRecord{
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeWechat,
						Amount:      10000,
					},
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      20000,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, expected payment.Payment) {
				actual, err := svc.FindPaymentByID(context.Background(), expected.ID)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := tc.newSvcFunc(t, ctrl)

			pmt, err := svc.CreatePayment(context.Background(), tc.pmt)

			tc.errRequireFunc(t, err)
			if err == nil {
				s.requirePayment(t, tc.pmt, pmt)
				tc.after(t, svc, pmt)
			}
		})
	}
}

func (s *PaymentModuleTestSuite) requirePayment(t *testing.T, expected, actual domain.Payment) {
	require.NotZero(t, actual.ID)
	require.NotZero(t, actual.SN)
	require.NotZero(t, actual.Ctime)
	actual.Records = slice.Map(actual.Records, func(idx int, src domain.PaymentRecord) domain.PaymentRecord {
		require.NotZero(t, src.PaymentID)
		require.Equal(t, actual.ID, src.PaymentID)
		require.NotZero(t, src.Status.ToUint8())
		require.Equal(t, actual.Status.ToUint8(), src.Status.ToUint8())
		src.PaymentID = 0
		src.Status = domain.PaymentStatus(0)
		return src
	})
	actual.ID = 0
	actual.SN = ""
	actual.Ctime = 0
	actual.Status = domain.PaymentStatus(0)
	require.ElementsMatch(t, expected.Records, actual.Records)
	expected.Records, actual.Records = nil, nil
	require.Equal(t, expected, actual)
}

func (s *PaymentModuleTestSuite) TestService_GetPaymentChannels() {
	t := s.T()
	svc := startup.InitService(nil, &credit.Module{}, nil)
	channels := svc.GetPaymentChannels(context.Background())
	require.Equal(t, []domain.PaymentChannel{
		{Type: domain.ChannelTypeCredit, Desc: "积分"},
		{Type: domain.ChannelTypeWechat, Desc: "微信"},
	}, channels)
}

func (s *PaymentModuleTestSuite) TestService_PayByID() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T, svc service.Service, pmt payment.Payment) int64
		pmt            payment.Payment
		newSvcFunc     func(t *testing.T, ctrl *gomock.Controller) service.Service
		errRequireFunc require.ErrorAssertionFunc
		after          func(t *testing.T, svc service.Service, expected payment.Payment)
	}{
		{
			name: "支付成功_仅积分支付",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200001,
				OrderSN:          "order-pay-200001",
				PayerID:          200001,
				OrderDescription: "月会员 * 1",
				TotalAmount:      990,
				Records: []domain.PaymentRecord{
					{
						Description: "月会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      990,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				evt := event.PaymentEvent{
					OrderSN: "order-pay-200001",
					PayerID: int64(200001),
					Status:  domain.PaymentStatusPaidSuccess.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(1), nil)
				mockCreditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

				return startup.InitService(mockProducer, &credit.Module{Svc: mockCreditSvc}, nil)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, expected payment.Payment) {
				t.Helper()
				actual, err := svc.FindPaymentByID(context.Background(), expected.ID)
				require.NoError(t, err)

				require.Equal(t, expected, actual)
				require.Equal(t, domain.PaymentStatusPaidSuccess, actual.Status)
				require.NotZero(t, actual.PaidAt)

				r, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidSuccess, r.Status)
				require.Equal(t, "1", r.PaymentNO3rd)
				require.NotZero(t, r.PaidAt)
			},
		},
		{
			name: "支付成功_仅积分支付_发送消息失败",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200006,
				OrderSN:          "order-pay-200006",
				PayerID:          200006,
				OrderDescription: "月会员 * 1",
				TotalAmount:      990,
				Records: []domain.PaymentRecord{
					{
						Description: "月会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      990,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				evt := event.PaymentEvent{
					OrderSN: "order-pay-200006",
					PayerID: int64(200006),
					Status:  domain.PaymentStatusPaidSuccess.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(errors.New("mock: 发送消息"))

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(4), nil)
				mockCreditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

				return startup.InitService(mockProducer, &credit.Module{Svc: mockCreditSvc}, nil)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, expected payment.Payment) {
				t.Helper()
				actual, err := svc.FindPaymentByID(context.Background(), expected.ID)
				require.NoError(t, err)

				require.Equal(t, expected, actual)
				require.Equal(t, domain.PaymentStatusPaidSuccess, actual.Status)
				require.NotZero(t, actual.PaidAt)

				r, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidSuccess, r.Status)
				require.Equal(t, "4", r.PaymentNO3rd)
				require.NotZero(t, r.PaidAt)
			},
		},
		{
			name: "支付失败_仅积分支付_预扣积分失败",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200004,
				OrderSN:          "order-pay-200004",
				PayerID:          200004,
				OrderDescription: "月会员 * 1",
				TotalAmount:      990,
				Records: []domain.PaymentRecord{
					{
						Description: "月会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      990,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(0), errors.New("mock: 积分不足"))

				return startup.InitService(nil, &credit.Module{Svc: mockCreditSvc}, nil)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
		{
			name: "支付失败_仅积分支付_确认扣减积分失败",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200005,
				OrderSN:          "order-pay-200005",
				PayerID:          200005,
				OrderDescription: "月会员 * 1",
				TotalAmount:      990,
				Records: []domain.PaymentRecord{
					{
						Description: "月会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      990,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(3), nil)
				mockCreditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("mock: 确认扣减积分失败"))
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

				return startup.InitService(nil, &credit.Module{Svc: mockCreditSvc}, nil)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
		{
			name: "支付失败_仅积分支付_支付金额非法",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200009,
				OrderSN:          "order-pay-200009",
				PayerID:          200009,
				OrderDescription: "月会员 * 1",
				TotalAmount:      990,
				Records: []domain.PaymentRecord{
					{
						Description: "月会员 * 1",
						Channel:     domain.ChannelTypeCredit,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
		{
			name: "支付失败_仅积分支付_支付ID非法",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				return int64(1000000)
			},
			pmt: payment.Payment{
				OrderID:          200015,
				OrderSN:          "order-pay-200015",
				PayerID:          200015,
				OrderDescription: "月会员 * 1",
				TotalAmount:      990,
				Records: []domain.PaymentRecord{
					{
						Description: "月会员 * 1",
						Channel:     domain.ChannelTypeCredit,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
		{
			name: "预支付成功_仅微信支付",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200002,
				OrderSN:          "order-pay-200002",
				PayerID:          200002,
				OrderDescription: "季会员 * 1",
				TotalAmount:      10000,
				Records: []domain.PaymentRecord{
					{
						Description: "季会员 * 1",
						Channel:     domain.ChannelTypeWechat,
						Amount:      10000,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockNativeAPI := wechatmocks.NewMockNativeAPIService(ctrl)
				codeURL := "code_url_wechat_only"
				mockNativeAPI.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(&native.PrepayResponse{
					CodeUrl: &codeURL,
				}, &core.APIResult{}, nil)

				return startup.InitService(nil, &credit.Module{}, mockNativeAPI)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, expected payment.Payment) {
				t.Helper()
				actual, err := svc.FindPaymentByID(context.Background(), expected.ID)
				require.NoError(t, err)

				expectedWechatCodeURL := "code_url_wechat_only"
				actual.Records[0].WechatCodeURL = expectedWechatCodeURL

				require.Equal(t, expected, actual)
				require.Equal(t, domain.PaymentStatusProcessing, actual.Status)
				require.Zero(t, actual.PaidAt)

				r, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, r.Status)
				require.Equal(t, expectedWechatCodeURL, r.WechatCodeURL)
				require.Zero(t, r.PaymentNO3rd)
				require.Zero(t, r.PaidAt)
			},
		},
		{
			name: "预支付失败_仅微信支付_获取二维码失败",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200007,
				OrderSN:          "order-pay-200007",
				PayerID:          200007,
				OrderDescription: "季会员 * 1",
				TotalAmount:      10000,
				Records: []domain.PaymentRecord{
					{
						Description: "季会员 * 1",
						Channel:     domain.ChannelTypeWechat,
						Amount:      10000,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockNativeAPI := wechatmocks.NewMockNativeAPIService(ctrl)
				mockNativeAPI.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(&native.PrepayResponse{}, &core.APIResult{}, errors.New("mock: 未知错误"))

				return startup.InitService(nil, &credit.Module{}, mockNativeAPI)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
		{
			name: "预支付失败_仅微信支付_支付金额非法",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200008,
				OrderSN:          "order-pay-200008",
				PayerID:          200008,
				OrderDescription: "季会员 * 1",
				TotalAmount:      10000,
				Records: []domain.PaymentRecord{
					{
						Description: "季会员 * 1",
						Channel:     domain.ChannelTypeWechat,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
		{
			name: "预支付失败_仅微信支付_支付ID非法",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				return int64(1000001)
			},
			pmt: payment.Payment{
				OrderID:          200016,
				OrderSN:          "order-pay-200016",
				PayerID:          200016,
				OrderDescription: "季会员 * 1",
				TotalAmount:      10000,
				Records: []domain.PaymentRecord{
					{
						Description: "季会员 * 1",
						Channel:     domain.ChannelTypeWechat,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
		{
			name: "预支付成功_混合支付",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200003,
				OrderSN:          "order-pay-200003",
				PayerID:          200003,
				OrderDescription: "年会员 * 1",
				TotalAmount:      30000,
				Records: []domain.PaymentRecord{
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeWechat,
						Amount:      10000,
					},
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      20000,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(2), nil)

				mockNativeAPI := wechatmocks.NewMockNativeAPIService(ctrl)
				codeURL := "code_url_wechat_and_credit"
				mockNativeAPI.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(&native.PrepayResponse{
					CodeUrl: &codeURL,
				}, &core.APIResult{}, nil)

				return startup.InitService(nil, &credit.Module{Svc: mockCreditSvc}, mockNativeAPI)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, expected payment.Payment) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), expected.ID)
				require.NoError(t, err)

				expectedWechatCodeURL := "code_url_wechat_and_credit"
				idx := slice.IndexFunc(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.NotEqual(t, -1, idx)
				actual.Records[idx].WechatCodeURL = expectedWechatCodeURL
				require.Equal(t, expected, actual)
				require.Equal(t, domain.PaymentStatusProcessing, actual.Status)
				require.Zero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, w.Status)
				require.Equal(t, expectedWechatCodeURL, w.WechatCodeURL)
				require.Zero(t, w.PaymentNO3rd)
				require.Zero(t, w.PaidAt)

				c, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, c.Status)
				require.Equal(t, "2", c.PaymentNO3rd)
				require.Zero(t, c.PaidAt)
			},
		},
		{
			name: "预支付失败_混合支付_积分支付金额非法",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200013,
				OrderSN:          "order-pay-200013",
				PayerID:          200013,
				OrderDescription: "年会员 * 1",
				TotalAmount:      30000,
				Records: []domain.PaymentRecord{
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeWechat,
						Amount:      10000,
					},
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeCredit,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
		{
			name: "预支付失败_混合支付_微信支付金额非法",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200012,
				OrderSN:          "order-pay-200012",
				PayerID:          200012,
				OrderDescription: "年会员 * 1",
				TotalAmount:      30000,
				Records: []domain.PaymentRecord{
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeWechat,
					},
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      20000,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(5), nil)
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

				return startup.InitService(nil, &credit.Module{Svc: mockCreditSvc}, nil)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
		{
			name: "预支付失败_混合支付_获取二维码失败",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200011,
				OrderSN:          "order-pay-200011",
				PayerID:          200011,
				OrderDescription: "年会员 * 1",
				TotalAmount:      30000,
				Records: []domain.PaymentRecord{
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeWechat,
						Amount:      10000,
					},
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      20000,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(6), nil)
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

				mockNativeAPI := wechatmocks.NewMockNativeAPIService(ctrl)
				mockNativeAPI.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(&native.PrepayResponse{}, &core.APIResult{}, errors.New("mock: 获取二维码失败"))

				return startup.InitService(nil, &credit.Module{Svc: mockCreditSvc}, mockNativeAPI)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
		{
			name: "预支付失败_混合支付_预扣积分失败",
			before: func(t *testing.T, svc service.Service, pmt payment.Payment) int64 {
				t.Helper()
				p, err := svc.CreatePayment(context.Background(), pmt)
				require.NoError(t, err)
				return p.ID
			},
			pmt: payment.Payment{
				OrderID:          200010,
				OrderSN:          "order-pay-200010",
				PayerID:          200010,
				OrderDescription: "年会员 * 1",
				TotalAmount:      30000,
				Records: []domain.PaymentRecord{
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeWechat,
						Amount:      10000,
					},
					{
						Description: "年会员 * 1",
						Channel:     domain.ChannelTypeCredit,
						Amount:      20000,
					},
				},
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(0), errors.New("mock: 预扣积分失败"))

				return startup.InitService(nil, &credit.Module{Svc: mockCreditSvc}, nil)
			},
			errRequireFunc: require.Error,
			after:          func(t *testing.T, svc service.Service, expected payment.Payment) {},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := tc.newSvcFunc(t, ctrl)
			pmtID := tc.before(t, svc, tc.pmt)

			pmt, err := svc.PayByID(context.Background(), pmtID)

			tc.errRequireFunc(t, err)
			if err == nil {
				tc.after(t, svc, pmt)
			}
		})
	}
}

const (
	TradeStateSuccess    = "SUCCESS"    // 成功
	TradeStatePayError   = "PAYERROR"   // 失败
	TradeStateClosed     = "CLOSED"     // 失败
	TradeStateRevoked    = "REVOKED"    // 失败
	TradeStateNotPay     = "NOTPAY"     // 忽略或关闭该支付记录
	TradeStateUserPaying = "USERPAYING" // 忽略或关闭该支付记录
	TradeStateRefund     = "REFUND"     // 忽略或关闭该支付记录
	TradeStateInvalid    = "INVALID"
)

func (s *PaymentModuleTestSuite) TestService_HandleWechatCallback() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T, svc service.Service) int64
		txn            *payments.Transaction
		newSvcFunc     func(t *testing.T, ctrl *gomock.Controller) service.Service
		errRequireFunc require.ErrorAssertionFunc
		after          func(t *testing.T, svc service.Service, pmtID int64)
	}{
		{
			name: "处理'支付成功'支付通知_仅微信支付",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          300001,
					OrderSN:          "order-callback-300001",
					PayerID:          300001,
					OrderDescription: "季会员 * 1",
					TotalAmount:      20000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				_, err = svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return pmt.ID
			},
			txn: &payments.Transaction{
				OutTradeNo:    core.String("order-callback-300001"),
				TransactionId: core.String("wechat-transaction-id-300001"),
				TradeState:    core.String(TradeStateSuccess),
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				evt := event.PaymentEvent{
					OrderSN: "order-callback-300001",
					PayerID: int64(300001),
					Status:  domain.PaymentStatusPaidSuccess.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_300001")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)

				return startup.InitService(mockProducer, &credit.Module{}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusPaidSuccess, actual.Status)
				require.Equal(t, "order-callback-300001", actual.OrderSN)
				require.NotZero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidSuccess, w.Status)
				require.Equal(t, "wechat-transaction-id-300001", w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.NotZero(t, w.PaidAt)
			},
		},
		{
			name: "处理'支付失败'支付通知_仅微信支付",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          300002,
					OrderSN:          "order-callback-300002",
					PayerID:          300002,
					OrderDescription: "季会员 * 1",
					TotalAmount:      20000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				_, err = svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return pmt.ID
			},
			txn: &payments.Transaction{
				OutTradeNo:    core.String("order-callback-300002"),
				TransactionId: core.String("wechat-transaction-id-300002"),
				TradeState:    core.String(TradeStateClosed),
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				evt := event.PaymentEvent{
					OrderSN: "order-callback-300002",
					PayerID: int64(300002),
					Status:  domain.PaymentStatusPaidFailed.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_300002")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)

				return startup.InitService(mockProducer, &credit.Module{}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusPaidFailed, actual.Status)
				require.Equal(t, "order-callback-300002", actual.OrderSN)
				require.Zero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidFailed, w.Status)
				require.Equal(t, "wechat-transaction-id-300002", w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.Zero(t, w.PaidAt)
			},
		},
		{
			name: "忽略'未支付'通知_仅微信支付",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          300003,
					OrderSN:          "order-callback-300003",
					PayerID:          300003,
					OrderDescription: "季会员 * 1",
					TotalAmount:      20000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				_, err = svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return pmt.ID
			},
			txn: &payments.Transaction{
				OutTradeNo:    core.String("order-callback-300003"),
				TransactionId: core.String("wechat-transaction-id-300003"),
				TradeState:    core.String(TradeStateNotPay),
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_300003")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)
				return startup.InitService(nil, &credit.Module{}, mockNativeAPIService)
			},
			errRequireFunc: require.Error,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusProcessing, actual.Status)
				require.Equal(t, "order-callback-300003", actual.OrderSN)
				require.Zero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, w.Status)
				require.Zero(t, w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.Zero(t, w.PaidAt)
			},
		},
		{
			name: "忽略'非法状态'通知_仅微信支付",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          300004,
					OrderSN:          "order-callback-300004",
					PayerID:          300004,
					OrderDescription: "季会员 * 1",
					TotalAmount:      20000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				_, err = svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return pmt.ID
			},
			txn: &payments.Transaction{
				OutTradeNo:    core.String("order-callback-300004"),
				TransactionId: core.String("wechat-transaction-id-300004"),
				TradeState:    core.String(TradeStateInvalid),
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_300004")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)
				return startup.InitService(nil, &credit.Module{}, mockNativeAPIService)
			},
			errRequireFunc: require.Error,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusProcessing, actual.Status)
				require.Equal(t, "order-callback-300004", actual.OrderSN)
				require.Zero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, w.Status)
				require.Zero(t, w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.Zero(t, w.PaidAt)
			},
		},

		{
			name: "处理'支付成功'支付通知_混合支付",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          300021,
					OrderSN:          "order-callback-300021",
					PayerID:          300021,
					OrderDescription: "季会员 * 1",
					TotalAmount:      30000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeCredit,
							Amount:      10000,
						},
					},
				})
				require.NoError(t, err)

				_, err = svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return pmt.ID
			},
			txn: &payments.Transaction{
				OutTradeNo:    core.String("order-callback-300021"),
				TransactionId: core.String("wechat-transaction-id-300021"),
				TradeState:    core.String(TradeStateSuccess),
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				payerID := int64(300021)
				evt := event.PaymentEvent{
					OrderSN: "order-callback-300021",
					PayerID: payerID,
					Status:  domain.PaymentStatusPaidSuccess.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				tid := int64(10)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(tid, nil)
				mockCreditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), payerID, tid).Return(nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_300021")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)

				return startup.InitService(mockProducer, &credit.Module{Svc: mockCreditSvc}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusPaidSuccess, actual.Status)
				require.Equal(t, "order-callback-300021", actual.OrderSN)
				require.NotZero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidSuccess, w.Status)
				require.Equal(t, "wechat-transaction-id-300021", w.PaymentNO3rd)
				require.NotZero(t, w.PaidAt)
				require.Zero(t, w.WechatCodeURL)

				c, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidSuccess, c.Status)
				require.Equal(t, "10", c.PaymentNO3rd)
				require.NotZero(t, c.PaidAt)
			},
		},
		{
			name: "处理'支付成功'支付通知_混合支付_确认扣减积分失败",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          300023,
					OrderSN:          "order-callback-300023",
					PayerID:          300023,
					OrderDescription: "季会员 * 1",
					TotalAmount:      30000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeCredit,
							Amount:      10000,
						},
					},
				})
				require.NoError(t, err)

				_, err = svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return pmt.ID
			},
			txn: &payments.Transaction{
				OutTradeNo:    core.String("order-callback-300023"),
				TransactionId: core.String("wechat-transaction-id-300023"),
				TradeState:    core.String(TradeStateSuccess),
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				payerID := int64(300023)
				evt := event.PaymentEvent{
					OrderSN: "order-callback-300023",
					PayerID: payerID,
					Status:  domain.PaymentStatusPaidSuccess.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockErr := errors.New("mock: 确认扣减积分失败")
				tid := int64(12)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(tid, nil)
				mockCreditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), payerID, tid).Return(mockErr)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_300023")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)

				return startup.InitService(mockProducer, &credit.Module{Svc: mockCreditSvc}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusPaidSuccess, actual.Status)
				require.Equal(t, "order-callback-300023", actual.OrderSN)
				require.NotZero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidSuccess, w.Status)
				require.Equal(t, "wechat-transaction-id-300023", w.PaymentNO3rd)
				require.NotZero(t, w.PaidAt)
				require.Zero(t, w.WechatCodeURL)

				c, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidSuccess, c.Status)
				require.Equal(t, "12", c.PaymentNO3rd)
				require.NotZero(t, c.PaidAt)
			},
		},
		{
			name: "处理'支付失败'支付通知_混合支付",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          300022,
					OrderSN:          "order-callback-300022",
					PayerID:          300022,
					OrderDescription: "季会员 * 1",
					TotalAmount:      30000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeCredit,
							Amount:      10000,
						},
					},
				})
				require.NoError(t, err)

				_, err = svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return pmt.ID
			},
			txn: &payments.Transaction{
				OutTradeNo:    core.String("order-callback-300022"),
				TransactionId: core.String("wechat-transaction-id-300022"),
				TradeState:    core.String(TradeStatePayError),
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				payerID := int64(300022)
				evt := event.PaymentEvent{
					OrderSN: "order-callback-300022",
					PayerID: payerID,
					Status:  domain.PaymentStatusPaidFailed.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				tid := int64(11)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(tid, nil)
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), payerID, tid).Return(nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_300022")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)

				return startup.InitService(mockProducer, &credit.Module{Svc: mockCreditSvc}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusPaidFailed, actual.Status)
				require.Equal(t, "order-callback-300022", actual.OrderSN)
				require.Zero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidFailed, w.Status)
				require.Equal(t, "wechat-transaction-id-300022", w.PaymentNO3rd)
				require.Zero(t, w.PaidAt)
				require.Zero(t, w.WechatCodeURL)

				c, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidFailed, c.Status)
				require.Equal(t, "11", c.PaymentNO3rd)
				require.Zero(t, c.PaidAt)
			},
		},
		{
			name: "处理'支付失败'支付通知_混合支付_取消预扣失败",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          300024,
					OrderSN:          "order-callback-300024",
					PayerID:          300024,
					OrderDescription: "季会员 * 1",
					TotalAmount:      30000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeCredit,
							Amount:      10000,
						},
					},
				})
				require.NoError(t, err)

				_, err = svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return pmt.ID
			},
			txn: &payments.Transaction{
				OutTradeNo:    core.String("order-callback-300024"),
				TransactionId: core.String("wechat-transaction-id-300024"),
				TradeState:    core.String(TradeStateRevoked),
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				payerID := int64(300024)
				evt := event.PaymentEvent{
					OrderSN: "order-callback-300024",
					PayerID: payerID,
					Status:  domain.PaymentStatusPaidFailed.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockErr := errors.New("mock: 取消预扣积分失败")
				tid := int64(13)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(tid, nil)
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), payerID, tid).Return(mockErr)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_300024")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)

				return startup.InitService(mockProducer, &credit.Module{Svc: mockCreditSvc}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusPaidFailed, actual.Status)
				require.Equal(t, "order-callback-300024", actual.OrderSN)
				require.Zero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidFailed, w.Status)
				require.Equal(t, "wechat-transaction-id-300024", w.PaymentNO3rd)
				require.Zero(t, w.PaidAt)
				require.Zero(t, w.WechatCodeURL)

				c, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidFailed, c.Status)
				require.Equal(t, "13", c.PaymentNO3rd)
				require.Zero(t, c.PaidAt)
			},
		},
		{
			name: "忽略'未支付'通知_混合支付",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          300025,
					OrderSN:          "order-callback-300025",
					PayerID:          300025,
					OrderDescription: "季会员 * 1",
					TotalAmount:      30000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeCredit,
							Amount:      10000,
						},
					},
				})
				require.NoError(t, err)

				_, err = svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return pmt.ID
			},
			txn: &payments.Transaction{
				OutTradeNo:    core.String("order-callback-300025"),
				TransactionId: core.String("wechat-transaction-id-300025"),
				TradeState:    core.String(TradeStateNotPay),
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(14), nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_300025")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)

				return startup.InitService(nil, &credit.Module{Svc: mockCreditSvc}, mockNativeAPIService)
			},
			errRequireFunc: require.Error,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusProcessing, actual.Status)
				require.Equal(t, "order-callback-300025", actual.OrderSN)
				require.Zero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, w.Status)
				require.Zero(t, w.PaymentNO3rd)
				require.Zero(t, w.PaidAt)
				require.Zero(t, w.WechatCodeURL)

				c, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, c.Status)
				require.Equal(t, "14", c.PaymentNO3rd)
				require.Zero(t, c.PaidAt)
			},
		},
		{
			name: "忽略'非法订单SN'通知_混合支付",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          300026,
					OrderSN:          "order-callback-300026",
					PayerID:          300026,
					OrderDescription: "季会员 * 1",
					TotalAmount:      30000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeCredit,
							Amount:      10000,
						},
					},
				})
				require.NoError(t, err)

				_, err = svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return pmt.ID
			},
			txn: &payments.Transaction{
				OutTradeNo:    core.String("order-callback-invalid-300026"),
				TransactionId: core.String("wechat-transaction-id-300026"),
				TradeState:    core.String(TradeStateSuccess),
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(15), nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_300026")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)

				return startup.InitService(nil, &credit.Module{Svc: mockCreditSvc}, mockNativeAPIService)
			},
			errRequireFunc: require.Error,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				actual, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusProcessing, actual.Status)
				require.Equal(t, "order-callback-300026", actual.OrderSN)
				require.Zero(t, actual.PaidAt)

				w, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, w.Status)
				require.Zero(t, w.PaymentNO3rd)
				require.Zero(t, w.PaidAt)
				require.Zero(t, w.WechatCodeURL)

				c, ok := slice.Find(actual.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, c.Status)
				require.Equal(t, "15", c.PaymentNO3rd)
				require.Zero(t, c.PaidAt)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := tc.newSvcFunc(t, ctrl)
			pmtID := tc.before(t, svc)
			err := svc.HandleWechatCallback(context.Background(), tc.txn)
			tc.errRequireFunc(t, err)
			tc.after(t, svc, pmtID)
		})
	}
}

func (s *PaymentModuleTestSuite) TestService_FindTimeoutPayments() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T, svc service.Service) int64
		newSvcFunc     func(t *testing.T, ctrl *gomock.Controller) service.Service
		offset         int
		limit          int
		ctime          int64
		errRequireFunc require.ErrorAssertionFunc
		after          func(t *testing.T, pmts []domain.Payment)
	}{
		{
			name: "查找超时支付成功_未支付状态",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				// 创建多个
				n := 3
				for i := 0; i < n; i++ {

					payerID := int64(400000 + i)

					_, err := svc.CreatePayment(context.Background(), domain.Payment{
						OrderID:          payerID,
						OrderSN:          fmt.Sprintf("order-timeout-%d", payerID),
						PayerID:          payerID,
						OrderDescription: "季会员 * 1",
						TotalAmount:      30000,
						Records: []domain.PaymentRecord{
							{
								Description: "季会员 * 1",
								Channel:     domain.ChannelTypeCredit,
								Amount:      30000,
							},
						},
					})
					require.NoError(t, err)
				}
				return int64(n)
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			offset:         0,
			limit:          3,
			ctime:          time.Now().Add(10 * time.Second).UnixMilli(),
			errRequireFunc: require.NoError,
			after: func(t *testing.T, pmts []domain.Payment) {
				t.Helper()
				for _, p := range pmts {
					require.Equal(t, domain.PaymentStatusUnpaid, p.Status)
					r, ok := slice.Find(p.Records, func(src domain.PaymentRecord) bool {
						return src.Channel == domain.ChannelTypeCredit
					})
					require.True(t, ok)
					require.Equal(t, domain.PaymentStatusUnpaid, r.Status)
				}
			},
		},
		{
			name: "查找超时支付成功_支付中状态",
			before: func(t *testing.T, svc service.Service) int64 {
				t.Helper()
				// 创建多个
				n := 5
				for i := 0; i < n; i++ {

					payerID := int64(400003 + i)

					pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
						OrderID:          payerID,
						OrderSN:          fmt.Sprintf("order-timeout-%d", payerID),
						PayerID:          payerID,
						OrderDescription: "季会员 * 1",
						TotalAmount:      30000,
						Records: []domain.PaymentRecord{
							{
								Description: "季会员 * 1",
								Channel:     domain.ChannelTypeWechat,
								Amount:      15000,
							},
							{
								Description: "季会员 * 1",
								Channel:     domain.ChannelTypeCredit,
								Amount:      15000,
							},
						},
					})
					require.NoError(t, err)

					_, err = svc.PayByID(context.Background(), pmt.ID)
					require.NoError(t, err)
				}
				return int64(n)
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(21), nil)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(22), nil)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(23), nil)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(24), nil)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(25), nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_40000xx")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil).Times(5)

				return startup.InitService(nil, &credit.Module{Svc: mockCreditSvc}, mockNativeAPIService)
			},
			offset:         0,
			limit:          2,
			ctime:          time.Now().Add(10 * time.Second).UnixMilli(),
			errRequireFunc: require.NoError,
			after: func(t *testing.T, pmts []domain.Payment) {
				t.Helper()
				for _, p := range pmts {
					require.Equal(t, domain.PaymentStatusProcessing, p.Status)
					c, ok := slice.Find(p.Records, func(src domain.PaymentRecord) bool {
						return src.Channel == domain.ChannelTypeCredit
					})
					require.True(t, ok)
					require.Equal(t, domain.PaymentStatusProcessing, c.Status)

					w, ok := slice.Find(p.Records, func(src domain.PaymentRecord) bool {
						return src.Channel == domain.ChannelTypeWechat
					})
					require.True(t, ok)
					require.Equal(t, domain.PaymentStatusProcessing, w.Status)
				}

			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := tc.newSvcFunc(t, ctrl)
			expectedTotal := tc.before(t, svc)
			pmts, total, err := svc.FindTimeoutPayments(context.Background(), tc.offset, tc.limit, tc.ctime)
			tc.errRequireFunc(t, err)
			if err == nil {
				require.Equal(t, expectedTotal, total)
				tc.after(t, pmts)
			}
		})
		s.TearDownTest()
	}

}

func (s *PaymentModuleTestSuite) TestService_CloseTimeoutPayment() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T, svc service.Service) []domain.Payment
		newSvcFunc     func(t *testing.T, ctrl *gomock.Controller) service.Service
		errRequireFunc require.ErrorAssertionFunc
		after          func(t *testing.T, svc service.Service, pmtID int64)
	}{
		{
			name: "关闭超时支付成功_未支付状态",
			before: func(t *testing.T, svc service.Service) []domain.Payment {
				t.Helper()
				// 创建多个
				n := 3
				pmts := make([]domain.Payment, 0, n)
				for i := 0; i < n; i++ {

					payerID := int64(400010 + i)

					pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
						OrderID:          payerID,
						OrderSN:          fmt.Sprintf("order-close-timeout-%d", payerID),
						PayerID:          payerID,
						OrderDescription: "季会员 * 1",
						TotalAmount:      30000,
						Records: []domain.PaymentRecord{
							{
								Description: "季会员 * 1",
								Channel:     domain.ChannelTypeCredit,
								Amount:      30000,
							},
						},
					})
					pmts = append(pmts, pmt)
					require.NoError(t, err)
				}
				return pmts
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				return startup.InitService(nil, &credit.Module{}, nil)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()
				p, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)
				require.Equal(t, domain.PaymentStatusTimeoutClosed, p.Status)
				r, ok := slice.Find(p.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusTimeoutClosed, r.Status)
			},
		},
		{
			name: "关闭超时支付成功_支付中状态",
			before: func(t *testing.T, svc service.Service) []domain.Payment {
				t.Helper()
				// 创建多个
				n := 6
				pmts := make([]domain.Payment, 0, n)
				for i := 0; i < n; i++ {

					payerID := int64(400020 + i)

					p, err := svc.CreatePayment(context.Background(), domain.Payment{
						OrderID:          payerID,
						OrderSN:          fmt.Sprintf("order-close-timeout-%d", payerID),
						PayerID:          payerID,
						OrderDescription: "季会员 * 1",
						TotalAmount:      30000,
						Records: []domain.PaymentRecord{
							{
								Description: "季会员 * 1",
								Channel:     domain.ChannelTypeWechat,
								Amount:      17000,
							},
							{
								Description: "季会员 * 1",
								Channel:     domain.ChannelTypeCredit,
								Amount:      13000,
							},
						},
					})
					require.NoError(t, err)

					pmt, err := svc.PayByID(context.Background(), p.ID)
					pmts = append(pmts, pmt)

					require.NoError(t, err)
				}
				return pmts
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(30), nil)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(31), nil)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(32), nil)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(33), nil)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(34), nil)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(35), nil)
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), int64(400020), int64(30)).Return(nil)
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), int64(400021), int64(31)).Return(nil)
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), int64(400022), int64(32)).Return(nil)
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), int64(400023), int64(33)).Return(nil)
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), int64(400024), int64(34)).Return(nil)
				mockCreditSvc.EXPECT().CancelDeductCredits(gomock.Any(), int64(400025), int64(35)).Return(nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_40020xx")}
				result := &core.APIResult{}
				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil).Times(6)

				return startup.InitService(nil, &credit.Module{Svc: mockCreditSvc}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()
				p, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusTimeoutClosed, p.Status)
				c, ok := slice.Find(p.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusTimeoutClosed, c.Status)
				require.Zero(t, c.PaidAt)

				w, ok := slice.Find(p.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusTimeoutClosed, w.Status)
				require.Zero(t, c.PaidAt)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := tc.newSvcFunc(t, ctrl)
			pmts := tc.before(t, svc)
			for _, pmt := range pmts {
				err := svc.CloseTimeoutPayment(context.Background(), pmt)
				tc.errRequireFunc(t, err)
				if err == nil {
					tc.after(t, svc, pmt.ID)
				}
			}
		})
		s.TearDownTest()
	}
}

func (s *PaymentModuleTestSuite) TestService_SyncWechatInfo() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T, svc service.Service) domain.Payment
		txn            *payments.Transaction
		newSvcFunc     func(t *testing.T, ctrl *gomock.Controller) service.Service
		errRequireFunc require.ErrorAssertionFunc
		after          func(t *testing.T, svc service.Service, pmtID int64)
	}{
		{
			name: "同步成功_微信支付_支付成功",
			before: func(t *testing.T, svc service.Service) domain.Payment {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          500001,
					OrderSN:          "order-sync-500001",
					PayerID:          500001,
					OrderDescription: "季会员 * 1",
					TotalAmount:      20000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				p, err := svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return p
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				orderSN := "order-sync-500001"
				evt := event.PaymentEvent{
					OrderSN: orderSN,
					PayerID: int64(500001),
					Status:  domain.PaymentStatusPaidSuccess.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_500001")}
				result := &core.APIResult{}

				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)
				req := native.QueryOrderByOutTradeNoRequest{OutTradeNo: core.String(orderSN), Mchid: core.String("MockMchID")}
				txn := &payments.Transaction{
					OutTradeNo:    core.String(orderSN),
					TransactionId: core.String("wechat-transaction-id-500001"),
					TradeState:    core.String(TradeStateSuccess),
				}
				mockNativeAPIService.EXPECT().QueryOrderByOutTradeNo(gomock.Any(), req).Return(txn, result, nil)

				return startup.InitService(mockProducer, &credit.Module{}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				expected, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusPaidSuccess, expected.Status)
				require.Equal(t, "order-sync-500001", expected.OrderSN)
				require.NotZero(t, expected.PaidAt)

				w, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidSuccess, w.Status)
				require.Equal(t, "wechat-transaction-id-500001", w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.NotZero(t, w.PaidAt)
			},
		},
		{
			name: "同步成功_微信支付_支付失败",
			before: func(t *testing.T, svc service.Service) domain.Payment {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          500002,
					OrderSN:          "order-sync-500002",
					PayerID:          500002,
					OrderDescription: "季会员 * 1",
					TotalAmount:      20000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				p, err := svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return p
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				orderSN := "order-sync-500002"
				evt := event.PaymentEvent{
					OrderSN: orderSN,
					PayerID: int64(500002),
					Status:  domain.PaymentStatusPaidFailed.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_500002")}
				result := &core.APIResult{}

				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)
				req := native.QueryOrderByOutTradeNoRequest{OutTradeNo: core.String(orderSN), Mchid: core.String("MockMchID")}
				txn := &payments.Transaction{
					OutTradeNo:    core.String(orderSN),
					TransactionId: core.String("wechat-transaction-id-500002"),
					TradeState:    core.String(TradeStatePayError),
				}
				mockNativeAPIService.EXPECT().QueryOrderByOutTradeNo(gomock.Any(), req).Return(txn, result, nil)

				return startup.InitService(mockProducer, &credit.Module{}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				expected, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusPaidFailed, expected.Status)
				require.Equal(t, "order-sync-500002", expected.OrderSN)
				require.Zero(t, expected.PaidAt)

				w, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidFailed, w.Status)
				require.Equal(t, "wechat-transaction-id-500002", w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.Zero(t, w.PaidAt)
			},
		},
		{
			name: "同步成功_微信支付_超时关闭",
			before: func(t *testing.T, svc service.Service) domain.Payment {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          500003,
					OrderSN:          "order-sync-500003",
					PayerID:          500003,
					OrderDescription: "季会员 * 1",
					TotalAmount:      20000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				p, err := svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return p
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_500003")}
				result := &core.APIResult{}

				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)
				orderSN := "order-sync-500003"
				req := native.QueryOrderByOutTradeNoRequest{OutTradeNo: core.String(orderSN), Mchid: core.String("MockMchID")}
				txn := &payments.Transaction{
					OutTradeNo:    core.String(orderSN),
					TransactionId: core.String("wechat-transaction-id-500003"),
					TradeState:    core.String(TradeStateUserPaying),
				}
				mockNativeAPIService.EXPECT().QueryOrderByOutTradeNo(gomock.Any(), req).Return(txn, result, nil)

				return startup.InitService(nil, &credit.Module{}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				expected, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusTimeoutClosed, expected.Status)
				require.Equal(t, "order-sync-500003", expected.OrderSN)
				require.Zero(t, expected.PaidAt)

				w, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusTimeoutClosed, w.Status)
				require.Equal(t, "wechat-transaction-id-500003", w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.Zero(t, w.PaidAt)
			},
		},
		{
			name: "同步失败_微信支付_向微信查询订单失败",
			before: func(t *testing.T, svc service.Service) domain.Payment {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          500004,
					OrderSN:          "order-sync-500004",
					PayerID:          500004,
					OrderDescription: "季会员 * 1",
					TotalAmount:      20000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				p, err := svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return p
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_500004")}
				result := &core.APIResult{}

				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)
				orderSN := "order-sync-500004"
				req := native.QueryOrderByOutTradeNoRequest{OutTradeNo: core.String(orderSN), Mchid: core.String("MockMchID")}
				txn := &payments.Transaction{}
				mockErr := errors.New("mock: 通过订单序列号查询微信订单失败")
				mockNativeAPIService.EXPECT().QueryOrderByOutTradeNo(gomock.Any(), req).Return(txn, result, mockErr)

				return startup.InitService(nil, &credit.Module{}, mockNativeAPIService)
			},
			errRequireFunc: require.Error,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				expected, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusProcessing, expected.Status)
				require.Equal(t, "order-sync-500004", expected.OrderSN)
				require.Zero(t, expected.PaidAt)

				w, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, w.Status)
				require.Zero(t, w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.Zero(t, w.PaidAt)
			},
		},
		{
			name: "同步失败_微信支付_交易状态非法",
			before: func(t *testing.T, svc service.Service) domain.Payment {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          500005,
					OrderSN:          "order-sync-500005",
					PayerID:          500005,
					OrderDescription: "季会员 * 1",
					TotalAmount:      20000,
					Records: []domain.PaymentRecord{
						{
							Description: "季会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				p, err := svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return p
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_500005")}
				result := &core.APIResult{}

				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)
				orderSN := "order-sync-500005"
				req := native.QueryOrderByOutTradeNoRequest{OutTradeNo: core.String(orderSN), Mchid: core.String("MockMchID")}
				txn := &payments.Transaction{
					OutTradeNo:    core.String(orderSN),
					TransactionId: core.String("wechat-transaction-id-500005"),
					TradeState:    core.String(TradeStateInvalid),
				}
				mockNativeAPIService.EXPECT().QueryOrderByOutTradeNo(gomock.Any(), req).Return(txn, result, nil)

				return startup.InitService(nil, &credit.Module{}, mockNativeAPIService)
			},
			errRequireFunc: require.Error,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				expected, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusProcessing, expected.Status)
				require.Equal(t, "order-sync-500005", expected.OrderSN)
				require.Zero(t, expected.PaidAt)

				w, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusProcessing, w.Status)
				require.Zero(t, w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.Zero(t, w.PaidAt)
			},
		},
		{
			name: "同步成功_混合支付_支付成功",
			before: func(t *testing.T, svc service.Service) domain.Payment {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          500011,
					OrderSN:          "order-sync-500011",
					PayerID:          500011,
					OrderDescription: "季会员 * 1",
					TotalAmount:      40000,
					Records: []domain.PaymentRecord{
						{
							Description: "年会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
						{
							Description: "年会员 * 1",
							Channel:     domain.ChannelTypeCredit,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				p, err := svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return p
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				orderSN := "order-sync-500011"
				payerID := int64(500011)
				evt := event.PaymentEvent{
					OrderSN: orderSN,
					PayerID: payerID,
					Status:  domain.PaymentStatusPaidSuccess.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockCreditService := creditmocks.NewMockService(ctrl)
				tid := int64(51)
				mockCreditService.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(tid, nil)
				mockCreditService.EXPECT().ConfirmDeductCredits(gomock.Any(), payerID, tid).Return(nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_500011")}
				result := &core.APIResult{}

				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)
				req := native.QueryOrderByOutTradeNoRequest{OutTradeNo: core.String(orderSN), Mchid: core.String("MockMchID")}
				txn := &payments.Transaction{
					OutTradeNo:    core.String(orderSN),
					TransactionId: core.String("wechat-transaction-id-500011"),
					TradeState:    core.String(TradeStateSuccess),
				}
				mockNativeAPIService.EXPECT().QueryOrderByOutTradeNo(gomock.Any(), req).Return(txn, result, nil)

				return startup.InitService(mockProducer, &credit.Module{Svc: mockCreditService}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				expected, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusPaidSuccess, expected.Status)
				require.Equal(t, "order-sync-500011", expected.OrderSN)
				require.NotZero(t, expected.PaidAt)

				w, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidSuccess, w.Status)
				require.Equal(t, "wechat-transaction-id-500011", w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.NotZero(t, w.PaidAt)

				c, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidSuccess, c.Status)
				require.Equal(t, "51", c.PaymentNO3rd)
				require.NotZero(t, c.PaidAt)

			},
		},
		{
			name: "同步成功_混合支付_支付失败",
			before: func(t *testing.T, svc service.Service) domain.Payment {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          500012,
					OrderSN:          "order-sync-500012",
					PayerID:          500012,
					OrderDescription: "季会员 * 1",
					TotalAmount:      40000,
					Records: []domain.PaymentRecord{
						{
							Description: "年会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
						{
							Description: "年会员 * 1",
							Channel:     domain.ChannelTypeCredit,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				p, err := svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return p
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
				orderSN := "order-sync-500012"
				payerID := int64(500012)
				evt := event.PaymentEvent{
					OrderSN: orderSN,
					PayerID: payerID,
					Status:  domain.PaymentStatusPaidFailed.ToUint8(),
				}
				mockProducer.EXPECT().Produce(gomock.Any(), evt).Return(nil)

				mockCreditService := creditmocks.NewMockService(ctrl)
				tid := int64(52)
				mockCreditService.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(tid, nil)
				mockCreditService.EXPECT().CancelDeductCredits(gomock.Any(), payerID, tid).Return(nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_500012")}
				result := &core.APIResult{}

				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)
				req := native.QueryOrderByOutTradeNoRequest{OutTradeNo: core.String(orderSN), Mchid: core.String("MockMchID")}
				txn := &payments.Transaction{
					OutTradeNo:    core.String(orderSN),
					TransactionId: core.String("wechat-transaction-id-500012"),
					TradeState:    core.String(TradeStateClosed),
				}
				mockNativeAPIService.EXPECT().QueryOrderByOutTradeNo(gomock.Any(), req).Return(txn, result, nil)

				return startup.InitService(mockProducer, &credit.Module{Svc: mockCreditService}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				expected, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusPaidFailed, expected.Status)
				require.Equal(t, "order-sync-500012", expected.OrderSN)
				require.Zero(t, expected.PaidAt)

				w, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidFailed, w.Status)
				require.Equal(t, "wechat-transaction-id-500012", w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.Zero(t, w.PaidAt)

				c, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusPaidFailed, c.Status)
				require.Equal(t, "52", c.PaymentNO3rd)
				require.Zero(t, c.PaidAt)

			},
		},
		{
			name: "同步成功_混合支付_超时关闭",
			before: func(t *testing.T, svc service.Service) domain.Payment {
				t.Helper()
				pmt, err := svc.CreatePayment(context.Background(), domain.Payment{
					OrderID:          500013,
					OrderSN:          "order-sync-500013",
					PayerID:          500013,
					OrderDescription: "季会员 * 1",
					TotalAmount:      40000,
					Records: []domain.PaymentRecord{
						{
							Description: "年会员 * 1",
							Channel:     domain.ChannelTypeWechat,
							Amount:      20000,
						},
						{
							Description: "年会员 * 1",
							Channel:     domain.ChannelTypeCredit,
							Amount:      20000,
						},
					},
				})
				require.NoError(t, err)

				p, err := svc.PayByID(context.Background(), pmt.ID)
				require.NoError(t, err)

				return p
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()
				orderSN := "order-sync-500013"
				payerID := int64(500013)

				mockCreditService := creditmocks.NewMockService(ctrl)
				tid := int64(53)
				mockCreditService.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(tid, nil)
				mockCreditService.EXPECT().CancelDeductCredits(gomock.Any(), payerID, tid).Return(nil)

				mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
				resp := &native.PrepayResponse{CodeUrl: core.String("wechat_code_url_500013")}
				result := &core.APIResult{}

				mockNativeAPIService.EXPECT().Prepay(gomock.Any(), gomock.Any()).Return(resp, result, nil)
				req := native.QueryOrderByOutTradeNoRequest{OutTradeNo: core.String(orderSN), Mchid: core.String("MockMchID")}
				txn := &payments.Transaction{
					OutTradeNo:    core.String(orderSN),
					TransactionId: core.String("wechat-transaction-id-500013"),
					TradeState:    core.String(TradeStateRefund),
				}
				mockNativeAPIService.EXPECT().QueryOrderByOutTradeNo(gomock.Any(), req).Return(txn, result, nil)

				return startup.InitService(nil, &credit.Module{Svc: mockCreditService}, mockNativeAPIService)
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, svc service.Service, pmtID int64) {
				t.Helper()

				expected, err := svc.FindPaymentByID(context.Background(), pmtID)
				require.NoError(t, err)

				require.Equal(t, domain.PaymentStatusTimeoutClosed, expected.Status)
				require.Equal(t, "order-sync-500013", expected.OrderSN)
				require.Zero(t, expected.PaidAt)

				w, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeWechat
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusTimeoutClosed, w.Status)
				require.Equal(t, "wechat-transaction-id-500013", w.PaymentNO3rd)
				require.Zero(t, w.WechatCodeURL)
				require.Zero(t, w.PaidAt)

				c, ok := slice.Find(expected.Records, func(src domain.PaymentRecord) bool {
					return src.Channel == domain.ChannelTypeCredit
				})
				require.True(t, ok)
				require.Equal(t, domain.PaymentStatusTimeoutClosed, c.Status)
				require.Equal(t, "53", c.PaymentNO3rd)
				require.Zero(t, c.PaidAt)

			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			svc := tc.newSvcFunc(t, ctrl)
			pmt := tc.before(t, svc)
			err := svc.SyncWechatInfo(context.Background(), pmt)
			tc.errRequireFunc(t, err)
			tc.after(t, svc, pmt.ID)
		})
	}

}

func (s *PaymentModuleTestSuite) TestJob_SyncWechatOrder() {
	t := s.T()
	t.Skip()

	// 创建超时数 < limit
	// 创建超时数 s.limit >= total
	// 创建超时数 s.limit < len(p) < total
	// 关闭超时成功_积分支付_超时关闭
}
