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
	"fmt"
	"testing"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/order"
	ordermocks "github.com/ecodeclub/webook/internal/order/mocks"
	"github.com/ecodeclub/webook/internal/payment"
	paymentmocks "github.com/ecodeclub/webook/internal/payment/mocks"
	"github.com/ecodeclub/webook/internal/recon/internal/job"
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

func (s *ReconModuleTestSuite) TestJob_Reconcile() {
	t := s.T()

	testCases := []struct {
		name           string
		limit          int
		newSvcFunc     func(t *testing.T, ctrl *gomock.Controller) service.Service
		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name:  "对账成功_订单为'支付中'状态_对应支付为'未支付'状态_关闭订单及支付",
			limit: 1,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				orders := []order.Order{
					{
						ID:      1,
						SN:      "order-sn-1",
						BuyerID: 1,
						Payment: order.Payment{
							ID: 1,
							SN: "payment-sn-1",
						},
						Status: order.StatusProcessing,
					},
					{
						ID:      2,
						SN:      "order-sn-2",
						BuyerID: 2,
						Payment: order.Payment{
							ID: 2,
							SN: "payment-sn-2",
						},
						Status: order.StatusProcessing,
					},
				}
				total := int64(len(orders))
				limit := 1
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[:limit], total, nil)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[limit:], total-int64(limit), nil)
				mockOrderSvc.EXPECT().FailOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				payments := []payment.Payment{
					{
						ID:               1,
						SN:               "payment-sn-1",
						PayerID:          1,
						OrderID:          1,
						OrderSN:          "order-sn-1",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusUnpaid,
						Records: []payment.Record{
							{
								PaymentID:    1,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusUnpaid,
							},
						},
					},
					{
						ID:               2,
						SN:               "payment-sn-2",
						PayerID:          2,
						OrderID:          2,
						OrderSN:          "order-sn-2",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusUnpaid,
						Records: []payment.Record{
							{
								PaymentID:    2,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusUnpaid,
							},
							{
								PaymentID:    2,
								PaymentNO3rd: "2",
								Channel:      payment.ChannelTypeCredit,
								PaidAt:       0,
								Status:       payment.StatusUnpaid,
							},
						},
					},
				}
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (payment.Payment, error) {
					p, ok := slice.Find(payments, func(src payment.Payment) bool {
						return src.ID == id
					})
					if !ok {
						return payment.Payment{}, fmt.Errorf("未配置支付记录: id:%d", id)
					}
					return p, nil
				}).Times(len(orders))
				mockPaymentSvc.EXPECT().SetPaymentStatusPaidFailed(gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))
				mockPaymentSvc.EXPECT().HandleCreditCallback(gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(6)

				return service.NewService(mockOrderSvc, mockPaymentSvc, nil, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.NoError,
		},
		{
			name:  "对账成功_订单为'支付中'状态_对应支付为'支付中'状态_关闭订单及支付_取消预扣积分",
			limit: 2,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				orders := []order.Order{
					{
						ID:      3,
						SN:      "order-sn-3",
						BuyerID: 3,
						Payment: order.Payment{
							ID: 3,
							SN: "payment-sn-3",
						},
						Status: order.StatusProcessing,
					},
					{
						ID:      4,
						SN:      "order-sn-4",
						BuyerID: 4,
						Payment: order.Payment{
							ID: 4,
							SN: "payment-sn-4",
						},
						Status: order.StatusProcessing,
					},
					{
						ID:      5,
						SN:      "order-sn-5",
						BuyerID: 5,
						Payment: order.Payment{
							ID: 5,
							SN: "payment-sn-5",
						},
						Status: order.StatusProcessing,
					},
				}
				total := int64(len(orders))
				limit := 2
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[:limit], total, nil)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[limit:], total-int64(limit), nil)
				mockOrderSvc.EXPECT().FailOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				payments := []payment.Payment{
					{
						ID:               3,
						SN:               "payment-sn-3",
						PayerID:          3,
						OrderID:          3,
						OrderSN:          "order-sn-3",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusProcessing,
						Records: []payment.Record{
							{
								PaymentID:    3,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusProcessing,
							},
						},
					},
					{
						ID:               4,
						SN:               "payment-sn-4",
						PayerID:          4,
						OrderID:          4,
						OrderSN:          "order-sn-4",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusProcessing,
						Records: []payment.Record{
							{
								PaymentID:    4,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusProcessing,
							},
							{
								PaymentID:    4,
								PaymentNO3rd: "4",
								Channel:      payment.ChannelTypeCredit,
								PaidAt:       0,
								Status:       payment.StatusProcessing,
							},
						},
					},
					{
						ID:               5,
						SN:               "payment-sn-5",
						PayerID:          5,
						OrderID:          5,
						OrderSN:          "order-sn-5",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusProcessing,
						Records: []payment.Record{
							{
								PaymentID:    5,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusProcessing,
							},
							{
								PaymentID:    5,
								PaymentNO3rd: "5",
								Channel:      payment.ChannelTypeCredit,
								PaidAt:       0,
								Status:       payment.StatusProcessing,
							},
						},
					},
				}
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (payment.Payment, error) {
					p, ok := slice.Find(payments, func(src payment.Payment) bool {
						return src.ID == id
					})
					if !ok {
						return payment.Payment{}, fmt.Errorf("未配置支付记录: id:%d", id)
					}
					return p, nil
				}).Times(len(orders))
				mockPaymentSvc.EXPECT().SetPaymentStatusPaidFailed(gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))
				mockPaymentSvc.EXPECT().HandleCreditCallback(gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(6)

				return service.NewService(mockOrderSvc, mockPaymentSvc, nil, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.NoError,
		},
		{
			name:  "对账成功_订单为'支付中'状态_对应支付为'支付成功'状态_修改订单为'支付成功'_确认预扣积分",
			limit: 2,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				orders := []order.Order{
					{
						ID:      6,
						SN:      "order-sn-6",
						BuyerID: 6,
						Payment: order.Payment{
							ID: 6,
							SN: "payment-sn-6",
						},
						Status: order.StatusProcessing,
					},
					{
						ID:      7,
						SN:      "order-sn-7",
						BuyerID: 7,
						Payment: order.Payment{
							ID: 7,
							SN: "payment-sn-7",
						},
						Status: order.StatusProcessing,
					},
					{
						ID:      8,
						SN:      "order-sn-8",
						BuyerID: 8,
						Payment: order.Payment{
							ID: 8,
							SN: "payment-sn-8",
						},
						Status: order.StatusProcessing,
					},
				}
				total := int64(len(orders))
				limit := 2
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[:limit], total, nil)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[limit:], total-int64(limit), nil)
				mockOrderSvc.EXPECT().SucceedOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				payments := []payment.Payment{
					{
						ID:               6,
						SN:               "payment-sn-6",
						PayerID:          6,
						OrderID:          6,
						OrderSN:          "order-sn-6",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusPaidSuccess,
						Records: []payment.Record{
							{
								PaymentID:    6,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusPaidSuccess,
							},
						},
					},
					{
						ID:               7,
						SN:               "payment-sn-7",
						PayerID:          7,
						OrderID:          7,
						OrderSN:          "order-sn-7",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusPaidSuccess,
						Records: []payment.Record{
							{
								PaymentID:    7,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusPaidSuccess,
							},
							{
								PaymentID:    7,
								PaymentNO3rd: "7",
								Channel:      payment.ChannelTypeCredit,
								PaidAt:       0,
								Status:       payment.StatusPaidSuccess,
							},
						},
					},
					{
						ID:               8,
						SN:               "payment-sn-8",
						PayerID:          8,
						OrderID:          8,
						OrderSN:          "order-sn-8",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusPaidSuccess,
						Records: []payment.Record{
							{
								PaymentID:    8,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusPaidSuccess,
							},
							{
								PaymentID:    8,
								PaymentNO3rd: "8",
								Channel:      payment.ChannelTypeCredit,
								PaidAt:       0,
								Status:       payment.StatusPaidSuccess,
							},
						},
					},
				}
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (payment.Payment, error) {
					p, ok := slice.Find(payments, func(src payment.Payment) bool {
						return src.ID == id
					})
					if !ok {
						return payment.Payment{}, fmt.Errorf("未配置支付记录: id:%d", id)
					}
					return p, nil
				}).Times(len(orders))
				mockPaymentSvc.EXPECT().HandleCreditCallback(gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(6)

				return service.NewService(mockOrderSvc, mockPaymentSvc, nil, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.NoError,
		},
		{
			name:  "对账成功_订单为'支付中'状态_对应支付为'支付失败'状态_修改订单为'支付失败'_取消预扣积分",
			limit: 1,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				orders := []order.Order{
					{
						ID:      9,
						SN:      "order-sn-9",
						BuyerID: 9,
						Payment: order.Payment{
							ID: 9,
							SN: "payment-sn-9",
						},
						Status: order.StatusProcessing,
					},
					{
						ID:      10,
						SN:      "order-sn-10",
						BuyerID: 10,
						Payment: order.Payment{
							ID: 10,
							SN: "payment-sn-10",
						},
						Status: order.StatusProcessing,
					},
				}
				total := int64(len(orders))
				limit := 1
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[:limit], total, nil)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[limit:], total-int64(limit), nil)
				mockOrderSvc.EXPECT().FailOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				payments := []payment.Payment{
					{
						ID:               9,
						SN:               "payment-sn-9",
						PayerID:          9,
						OrderID:          9,
						OrderSN:          "order-sn-9",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusPaidFailed,
						Records: []payment.Record{
							{
								PaymentID:    9,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusPaidFailed,
							},
						},
					},
					{
						ID:               10,
						SN:               "payment-sn-10",
						PayerID:          10,
						OrderID:          10,
						OrderSN:          "order-sn-10",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusPaidFailed,
						Records: []payment.Record{
							{
								PaymentID:    10,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusPaidFailed,
							},
							{
								PaymentID:    10,
								PaymentNO3rd: "10",
								Channel:      payment.ChannelTypeCredit,
								PaidAt:       0,
								Status:       payment.StatusPaidFailed,
							},
						},
					},
				}
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (payment.Payment, error) {
					p, ok := slice.Find(payments, func(src payment.Payment) bool {
						return src.ID == id
					})
					if !ok {
						return payment.Payment{}, fmt.Errorf("未配置支付记录: id:%d", id)
					}
					return p, nil
				}).Times(len(orders))
				mockPaymentSvc.EXPECT().HandleCreditCallback(gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(6)

				return service.NewService(mockOrderSvc, mockPaymentSvc, nil, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.NoError,
		},
		{
			name:  "对账失败_查找订单失败",
			limit: 1,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockErr := fmt.Errorf("mock: 查找订单失败")
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, int64(0), mockErr)

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(6)

				return service.NewService(mockOrderSvc, nil, nil, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.Error,
		},
		{
			name:  "对账失败_查找支付失败",
			limit: 1,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				orders := []order.Order{
					{
						ID:      11,
						SN:      "order-sn-11",
						BuyerID: 11,
						Payment: order.Payment{
							ID: 11,
							SN: "payment-sn-11",
						},
						Status: order.StatusProcessing,
					},
				}
				total := int64(len(orders))
				limit := 1
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[:limit], total, nil)

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				mockErr := fmt.Errorf("mock: 查找支付失败")
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).Return(payment.Payment{}, mockErr).Times(len(orders))

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(6)

				return service.NewService(mockOrderSvc, mockPaymentSvc, nil, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.NoError,
		},
		{
			name:  "对账失败_设置支付状态为支付失败状态失败",
			limit: 1,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				orders := []order.Order{
					{
						ID:      12,
						SN:      "order-sn-12",
						BuyerID: 12,
						Payment: order.Payment{
							ID: 12,
							SN: "payment-sn-12",
						},
						Status: order.StatusProcessing,
					},
				}
				total := int64(len(orders))
				limit := 1
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[:limit], total, nil)

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				payments := []payment.Payment{
					{
						ID:               12,
						SN:               "payment-sn-12",
						PayerID:          12,
						OrderID:          12,
						OrderSN:          "order-sn-12",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusUnpaid,
						Records: []payment.Record{
							{
								PaymentID:    12,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusUnpaid,
							},
						},
					},
				}
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (payment.Payment, error) {
					p, ok := slice.Find(payments, func(src payment.Payment) bool {
						return src.ID == id
					})
					if !ok {
						return payment.Payment{}, fmt.Errorf("未配置支付记录: id:%d", id)
					}
					return p, nil
				}).Times(len(orders))
				mockErr := fmt.Errorf("mock: 设置支付状态为支付失败状态失败")
				mockPaymentSvc.EXPECT().SetPaymentStatusPaidFailed(gomock.Any(), gomock.Any()).Return(mockErr).Times(len(orders))

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(6)

				return service.NewService(mockOrderSvc, mockPaymentSvc, nil, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.NoError,
		},
		{
			name:  "对账失败_确认或取消预扣积分失败",
			limit: 1,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				orders := []order.Order{
					{
						ID:      13,
						SN:      "order-sn-13",
						BuyerID: 13,
						Payment: order.Payment{
							ID: 13,
							SN: "payment-sn-13",
						},
						Status: order.StatusProcessing,
					},
				}
				total := int64(len(orders))
				limit := 1
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[:limit], total, nil)

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				payments := []payment.Payment{
					{
						ID:               13,
						SN:               "payment-sn-13",
						PayerID:          13,
						OrderID:          13,
						OrderSN:          "order-sn-13",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusUnpaid,
						Records: []payment.Record{
							{
								PaymentID:    13,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusUnpaid,
							},
						},
					},
				}
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (payment.Payment, error) {
					p, ok := slice.Find(payments, func(src payment.Payment) bool {
						return src.ID == id
					})
					if !ok {
						return payment.Payment{}, fmt.Errorf("未配置支付记录: id:%d", id)
					}
					return p, nil
				}).Times(len(orders))
				mockPaymentSvc.EXPECT().SetPaymentStatusPaidFailed(gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))
				mockErr := fmt.Errorf("mock: 确认/取消预扣积分失败")
				mockPaymentSvc.EXPECT().HandleCreditCallback(gomock.Any(), gomock.Any()).Return(mockErr).Times(len(orders))

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(len(orders))

				return service.NewService(mockOrderSvc, mockPaymentSvc, nil, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.NoError,
		},
		{
			name:  "对账失败_修改订单状态为支付成功状态失败",
			limit: 1,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				orders := []order.Order{
					{
						ID:      14,
						SN:      "order-sn-14",
						BuyerID: 14,
						Payment: order.Payment{
							ID: 14,
							SN: "payment-sn-14",
						},
						Status: order.StatusProcessing,
					},
				}
				total := int64(len(orders))
				limit := 1
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[:limit], total, nil)
				mockErr := fmt.Errorf("mock: 设置订单为支付成功失败")
				mockOrderSvc.EXPECT().SucceedOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockErr).Times(len(orders))

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				payments := []payment.Payment{
					{
						ID:               14,
						SN:               "payment-sn-14",
						PayerID:          14,
						OrderID:          14,
						OrderSN:          "order-sn-14",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusPaidSuccess,
						Records: []payment.Record{
							{
								PaymentID:    14,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusPaidSuccess,
							},
						},
					},
				}
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (payment.Payment, error) {
					p, ok := slice.Find(payments, func(src payment.Payment) bool {
						return src.ID == id
					})
					if !ok {
						return payment.Payment{}, fmt.Errorf("未配置支付记录: id:%d", id)
					}
					return p, nil
				}).Times(len(orders))
				mockPaymentSvc.EXPECT().HandleCreditCallback(gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(len(orders))

				return service.NewService(mockOrderSvc, mockPaymentSvc, nil, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.NoError,
		},
		{
			name:  "对账失败_修改订单状体为支付失败状态失败",
			limit: 1,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				orders := []order.Order{
					{
						ID:      15,
						SN:      "order-sn-15",
						BuyerID: 15,
						Payment: order.Payment{
							ID: 15,
							SN: "payment-sn-15",
						},
						Status: order.StatusProcessing,
					},
				}
				total := int64(len(orders))
				limit := 1
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[:limit], total, nil)
				mockErr := fmt.Errorf("mock: 设置订单为支付失败状态失败")
				mockOrderSvc.EXPECT().FailOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockErr).Times(len(orders))

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				payments := []payment.Payment{
					{
						ID:               15,
						SN:               "payment-sn-15",
						PayerID:          15,
						OrderID:          15,
						OrderSN:          "order-sn-15",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusPaidFailed,
						Records: []payment.Record{
							{
								PaymentID:    15,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusPaidFailed,
							},
						},
					},
				}
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (payment.Payment, error) {
					p, ok := slice.Find(payments, func(src payment.Payment) bool {
						return src.ID == id
					})
					if !ok {
						return payment.Payment{}, fmt.Errorf("未配置支付记录: id:%d", id)
					}
					return p, nil
				}).Times(len(orders))
				mockPaymentSvc.EXPECT().HandleCreditCallback(gomock.Any(), gomock.Any()).Return(nil).Times(len(orders))

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(len(orders))

				return service.NewService(mockOrderSvc, mockPaymentSvc, nil, initialInterval, maxInterval, maxRetries)
			},
			errRequireFunc: require.NoError,
		},
		{
			name:  "对账失败_超过最大重试次数",
			limit: 1,
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller) service.Service {
				t.Helper()

				orders := []order.Order{
					{
						ID:      16,
						SN:      "order-sn-16",
						BuyerID: 16,
						Payment: order.Payment{
							ID: 16,
							SN: "payment-sn-16",
						},
						Status: order.StatusProcessing,
					},
				}
				total := int64(len(orders))
				limit := 1
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().FindTimeoutOrders(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(orders[:limit], total, nil)
				mockErr := fmt.Errorf("mock: 设置订单为支付失败状态失败")
				mockOrderSvc.EXPECT().FailOrder(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockErr).AnyTimes()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				payments := []payment.Payment{
					{
						ID:               16,
						SN:               "payment-sn-16",
						PayerID:          16,
						OrderID:          16,
						OrderSN:          "order-sn-16",
						OrderDescription: "",
						PaidAt:           0,
						Status:           payment.StatusPaidFailed,
						Records: []payment.Record{
							{
								PaymentID:    16,
								PaymentNO3rd: "",
								Channel:      payment.ChannelTypeWechat,
								PaidAt:       0,
								Status:       payment.StatusPaidFailed,
							},
						},
					},
				}
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (payment.Payment, error) {
					p, ok := slice.Find(payments, func(src payment.Payment) bool {
						return src.ID == id
					})
					if !ok {
						return payment.Payment{}, fmt.Errorf("未配置支付记录: id:%d", id)
					}
					return p, nil
				}).AnyTimes()
				mockPaymentSvc.EXPECT().HandleCreditCallback(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

				initialInterval := 100 * time.Millisecond
				maxInterval := 1 * time.Second
				maxRetries := int32(len(orders))

				return service.NewService(mockOrderSvc, mockPaymentSvc, nil, initialInterval, maxInterval, maxRetries)
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
			j := job.NewSyncPaymentAndOrderJob(svc, 0, 0, tc.limit)
			require.NotZero(t, j.Name())
			err := j.Run(context.Background())
			tc.errRequireFunc(t, err)
		})
	}
}
