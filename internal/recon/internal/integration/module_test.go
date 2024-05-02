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
	"time"

	creditmocks "github.com/ecodeclub/webook/internal/credit/mocks"
	ordermocks "github.com/ecodeclub/webook/internal/order/mocks"
	paymentmocks "github.com/ecodeclub/webook/internal/payment/mocks"
	"github.com/ecodeclub/webook/internal/recon/internal/service"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestReconModule(t *testing.T) {
	suite.Run(t, new(ReconModuleTestSuite))
}

type ReconModuleTestSuite struct {
	suite.Suite
}

func (s *ReconModuleTestSuite) TestService_Reconcile() {
	t := s.T()
	t.Skip()

	testCases := []struct {
		name           string
		before         func(t *testing.T)
		offset         int
		limit          int
		ctime          int64
		newSvcFunc     func(t *testing.T, ctrl *gomock.Controller) service.Service
		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name: "对账成功_关闭订单及支付_订单处于未支付状态_支付处于未支付状态",
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				mockCreditSvc := creditmocks.NewMockService(ctrl)

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(6)

				return service.NewService(mockOrderSvc, mockPaymentSvc, mockCreditSvc, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.NoError,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			svc := tc.newSvcFunc(t, ctrl)
			err := svc.Reconcile(context.Background(), tc.offset, tc.limit, tc.ctime)
			tc.errRequireFunc(t, err)
		})
	}
}
