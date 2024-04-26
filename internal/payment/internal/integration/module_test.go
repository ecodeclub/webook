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
	"testing"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/credit"
	creditmocks "github.com/ecodeclub/webook/internal/credit/mocks"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	evtmocks "github.com/ecodeclub/webook/internal/payment/internal/event/mocks"
	startup "github.com/ecodeclub/webook/internal/payment/internal/integration/setup"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
	wechatmocks "github.com/ecodeclub/webook/internal/payment/internal/service/wechat/mocks"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
				mockProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(1), nil)
				mockCreditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

				return startup.InitService(mockProducer, &credit.Module{Svc: mockCreditSvc}, nil)
			},
			errRequireFunc: func(t require.TestingT, err error, i ...interface{}) {
				require.NoError(t, err)
			},
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
				require.NotZero(t, r.PaymentNO3rd)
				require.NotZero(t, r.PaidAt)
			},
		},
		// 支付成功_仅积分支付_状态改变
		// 支付成功_仅微信支付_状态改变_返回二维码
		// 支付成功_混合支付_状态改变_返回二维码

		// 支付失败_积分不足
		// 支付失败_混合支付失败_积分不足
		// 支付失败_混合支付失败_微信返回二维码失败
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

func (s *PaymentModuleTestSuite) TestHandler_Callback() {
	t := s.T()
	t.Skip()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProducer := evtmocks.NewMockPaymentEventProducer(ctrl)
	mockProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(nil)

	mockCreditSvc := creditmocks.NewMockService(ctrl)
	mockCreditSvc.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(1), nil)
	mockCreditSvc.EXPECT().ConfirmDeductCredits(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	mockNativeAPIService := wechatmocks.NewMockNativeAPIService(ctrl)
	mockNotifyHandler := wechatmocks.NewMockNotifyHandler(ctrl)

	handler := startup.InitHandler(
		mockProducer,
		&credit.Module{Svc: mockCreditSvc},
		mockNativeAPIService,
		mockNotifyHandler,
	)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: testUID,
		}))
	})
	handler.PublicRoutes(server.Engine)
}
