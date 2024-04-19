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
	"time"

	creditmocks "github.com/ecodeclub/webook/internal/credit/mocks"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/event"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
	credit2 "github.com/ecodeclub/webook/internal/payment/internal/service/credit"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const testUID = int64(789)

type CreditPaymentServiceTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	dao    dao.PaymentDAO
	repo   repository.PaymentRepository
}

func (s *CreditPaymentServiceTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	err := dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewPaymentGORMDAO(s.db)
	s.repo = repository.NewPaymentRepository(s.dao)
}

func (s *CreditPaymentServiceTestSuite) TearDownSuite() {
	// err := s.db.Exec("DROP TABLE `payments`").Error
	// require.NoError(s.T(), err)
	// err = s.db.Exec("DROP TABLE `payment_records`").Error
	// require.NoError(s.T(), err)
}

func (s *CreditPaymentServiceTestSuite) TearDownTest() {
	// err := s.db.Exec("TRUNCATE TABLE `payments`").Error
	// require.NoError(s.T(), err)
	// err = s.db.Exec("TRUNCATE TABLE `payment_records`").Error
	// require.NoError(s.T(), err)
}

func (s *CreditPaymentServiceTestSuite) TestPay() {
	t := s.T()

	t.Run("成功", func(t *testing.T) {
		t.Skip()
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockedCreditService := creditmocks.NewMockService(ctrl)
		paymentNo3rd := int64(1)
		mockedCreditService.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(paymentNo3rd, int64(10), nil)
		mockedCreditService.EXPECT().ConfirmDeductCredits(gomock.Any(), testUID, paymentNo3rd).Return(nil)
		// mockedCreditService.EXPECT().CancelDeductCredits(gomock.Any(), paymentNo3rd).Return(nil)

		paymentDDLFunc := func() int64 {
			return time.Now().Add(1 * time.Minute).UnixMilli()
		}
		producer := &fakeProducer{}
		svc := credit2.NewCreditPaymentService(mockedCreditService, s.repo, producer, paymentDDLFunc, sequencenumber.NewGenerator(), elog.DefaultLogger)

		pmt, err := svc.Pay(context.Background(), domain.Payment{
			OrderID:          200001,
			OrderSN:          "OrderSN-Payment-credit-001",
			PayerID:          testUID,
			OrderDescription: "月会员 * 1",
			TotalAmount:      990,
			Records: []domain.PaymentRecord{
				{
					Channel: domain.ChannelTypeCredit,
					Amount:  990,
				},
			},
		})
		require.NoError(t, err)
		require.NotZero(t, pmt.ID)
		require.NotZero(t, pmt.SN)

		require.Equal(t, "OrderSN-Payment-credit-001", producer.paymentEvents[0].OrderSN)
		require.Equal(t, int64(domain.PaymentStatusPaid), producer.paymentEvents[0].Status)
	})

	t.Run("失败_积分不足预扣失败", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockedCreditService := creditmocks.NewMockService(ctrl)
		mockedCreditService.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(int64(0), errors.New("预扣积分失败")).AnyTimes()

		paymentDDLFunc := func() int64 {
			return time.Now().Add(1 * time.Minute).UnixMilli()
		}
		svc := credit2.NewCreditPaymentService(mockedCreditService, s.repo, &fakeProducer{}, paymentDDLFunc, sequencenumber.NewGenerator(), elog.DefaultLogger)

		pmt, err := svc.Pay(context.Background(), domain.Payment{
			OrderID:          200002,
			OrderSN:          "OrderSN-Payment-credit-002",
			PayerID:          testUID,
			OrderDescription: "月会员 * 1",
			TotalAmount:      990,
			Records: []domain.PaymentRecord{
				{
					Channel: domain.ChannelTypeCredit,
					Amount:  990,
				},
			},
		})
		require.Error(t, err)
		require.Zero(t, pmt)
	})

	t.Run("失败_确认预扣失败", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockedCreditService := creditmocks.NewMockService(ctrl)
		expectedErr := errors.New("确认预扣积分失败")
		paymentNo3rd := int64(1)
		mockedCreditService.EXPECT().TryDeductCredits(gomock.Any(), gomock.Any()).Return(paymentNo3rd, nil)
		mockedCreditService.EXPECT().ConfirmDeductCredits(gomock.Any(), gomock.Any(), gomock.Any()).Return(expectedErr).AnyTimes()
		mockedCreditService.EXPECT().CancelDeductCredits(gomock.Any(), testUID, paymentNo3rd).Return(nil)

		paymentDDLFunc := func() int64 {
			return time.Now().Add(1 * time.Minute).UnixMilli()
		}
		svc := credit2.NewCreditPaymentService(mockedCreditService, s.repo, &fakeProducer{}, paymentDDLFunc, sequencenumber.NewGenerator(), elog.DefaultLogger)

		pmt, err := svc.Pay(context.Background(), domain.Payment{
			OrderID:          200003,
			OrderSN:          "OrderSN-Payment-credit-003",
			PayerID:          testUID,
			OrderDescription: "月会员 * 1",
			TotalAmount:      990,
			Records: []domain.PaymentRecord{
				{
					Channel: domain.ChannelTypeCredit,
					Amount:  990,
				},
			},
		})
		require.Error(t, err)
		require.Zero(t, pmt)
	})
}

type fakeProducer struct {
	paymentEvents []event.PaymentEvent
}

func (f *fakeProducer) Produce(_ context.Context, evt event.PaymentEvent) error {
	f.paymentEvents = append(f.paymentEvents, evt)
	return nil
}

func TestCreditPaymentServiceTestSuite(t *testing.T) {
	suite.Run(t, new(CreditPaymentServiceTestSuite))
}
