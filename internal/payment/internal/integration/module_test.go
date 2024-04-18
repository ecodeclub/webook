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
	"testing"
	"time"

	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/credit"
	creditmocks "github.com/ecodeclub/webook/internal/credit/mocks"
	"github.com/ecodeclub/webook/internal/payment"
	evtmocks "github.com/ecodeclub/webook/internal/payment/internal/event/mocks"
	startup "github.com/ecodeclub/webook/internal/payment/internal/integration/setup"
	wechatmocks "github.com/ecodeclub/webook/internal/payment/internal/service/wechat/mocks"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const testUID = int64(789)

func TestPaymentModule(t *testing.T) {
	suite.Run(t, new(PaymentModuleTestSuite))
}

type PaymentModuleTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	module *payment.Module
	ctrl   *gomock.Controller
}

func (s *PaymentModuleTestSuite) SetupSuite() {
	s.ctrl = gomock.NewController(s.T())

	s.module = startup.InitModule(
		s.getMockProducer(),
		s.paymentDDLFunc(),
		s.getMockCreditService(),
		s.getMockNotifyHandler(),
		s.getMockNativeAPIService())

	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: testUID,
		}))
	})
	s.module.Hdl.PublicRoutes(server.Engine)
	s.server = server
	s.db = testioc.InitDB()
}

func (s *PaymentModuleTestSuite) getMockNativeAPIService() *wechatmocks.MockNativeAPIService {
	nativeAPI := wechatmocks.NewMockNativeAPIService(s.ctrl)
	return nativeAPI
}

func (s *PaymentModuleTestSuite) getMockNotifyHandler() *wechatmocks.MockNotifyHandler {
	notifyHandler := wechatmocks.NewMockNotifyHandler(s.ctrl)
	return notifyHandler
}

func (s *PaymentModuleTestSuite) getMockProducer() *evtmocks.MockPaymentEventProducer {
	mockedPayment := evtmocks.NewMockPaymentEventProducer(s.ctrl)
	return mockedPayment
}

func (s *PaymentModuleTestSuite) getMockCreditService() *credit.Module {
	creditModule := &credit.Module{Svc: creditmocks.NewMockService(s.ctrl)}
	return creditModule
}

func (s *PaymentModuleTestSuite) paymentDDLFunc() func() int64 {
	return func() int64 {
		return time.Now().Add(time.Minute * 30).UnixMilli()
	}
}

func (s *PaymentModuleTestSuite) getCreditMockService() *creditmocks.MockService {
	mockedCreditSvc := creditmocks.NewMockService(s.ctrl)
	mockedCreditSvc.EXPECT().GetCreditsByUID(gomock.Any(), testUID).AnyTimes().Return(credit.Credit{
		TotalAmount: 1000,
	}, nil)
	return mockedCreditSvc
}

func (s *PaymentModuleTestSuite) TearDownSuite() {
	// err := s.db.Exec("DROP TABLE `payments`").Error
	// require.NoError(s.T(), err)
	// err = s.db.Exec("DROP TABLE `payment_records`").Error
	// require.NoError(s.T(), err)

	s.ctrl.Finish()
}

func (s *PaymentModuleTestSuite) TearDownTest() {
	// err := s.db.Exec("TRUNCATE TABLE `payments`").Error
	// require.NoError(s.T(), err)
	// err = s.db.Exec("TRUNCATE TABLE `payment_records`").Error
	// require.NoError(s.T(), err)
}

func (s *PaymentModuleTestSuite) TestService_CreatePayment_CreditOnly() {
	t := s.T()
	t.Skip()

}
