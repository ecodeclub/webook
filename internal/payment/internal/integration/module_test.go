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
	"testing"

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

const testUID = int64(789)

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
		require.NotZero(t, src.Status.ToUnit8())
		require.Equal(t, actual.Status.ToUnit8(), src.Status.ToUnit8())
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
					Status:  domain.PaymentStatusPaidSuccess.ToUnit8(),
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
					Status:  domain.PaymentStatusPaidSuccess.ToUnit8(),
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

func (s *PaymentModuleTestSuite) TestService_HandleWechatCallback() {
	t := s.T()

	const (
		TradeStateSuccess  = "SUCCESS"  // 成功
		TradeStatePayError = "PAYERROR" // 失败
		TradeStateClosed   = "CLOSED"   // 失败
		TradeStateRevoked  = "REVOKED"  // 失败
		TradeStateNotPay   = "NOTPAY"   // 忽略, 同样还有USERPAYING和REFUND,三者属于同一等价类划分
		TradeStateInvalid  = "INVALID"
	)

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
					Status:  domain.PaymentStatusPaidSuccess.ToUnit8(),
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
					Status:  domain.PaymentStatusPaidFailed.ToUnit8(),
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
					Status:  domain.PaymentStatusPaidSuccess.ToUnit8(),
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
					Status:  domain.PaymentStatusPaidSuccess.ToUnit8(),
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
					Status:  domain.PaymentStatusPaidFailed.ToUnit8(),
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
					Status:  domain.PaymentStatusPaidFailed.ToUnit8(),
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

func (s *PaymentModuleTestSuite) TestJob_SyncWechatOrder() {
	t := s.T()
	t.Skip()
}
