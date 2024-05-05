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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit"
	creditmocks "github.com/ecodeclub/webook/internal/credit/mocks"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/order/internal/domain"
	"github.com/ecodeclub/webook/internal/order/internal/errs"
	"github.com/ecodeclub/webook/internal/order/internal/event"
	"github.com/ecodeclub/webook/internal/order/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/order/internal/job"
	"github.com/ecodeclub/webook/internal/order/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/order/internal/web"
	"github.com/ecodeclub/webook/internal/payment"
	paymentmocks "github.com/ecodeclub/webook/internal/payment/mocks"
	"github.com/ecodeclub/webook/internal/product"
	productmocks "github.com/ecodeclub/webook/internal/product/mocks"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	testUID = int64(234)
)

func TestOrderModule(t *testing.T) {
	suite.Run(t, new(OrderModuleTestSuite))
}

type OrderModuleTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	mq     mq.MQ
	cache  ecache.Cache
	dao    dao.OrderDAO
	svc    order.Service
}

func (s *OrderModuleTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	err := dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewOrderGORMDAO(s.db)
	s.svc = order.InitService(s.db)
	s.mq = testioc.InitMQ()
	s.cache = testioc.InitCache()
}

func (s *OrderModuleTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `orders`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `order_items`").Error
	require.NoError(s.T(), err)
}

func (s *OrderModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `orders`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `order_items`").Error
	require.NoError(s.T(), err)
}

func (s *OrderModuleTestSuite) newGinServer(handler *web.Handler) *egin.Component {
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: testUID,
		}))
	})

	handler.PrivateRoutes(server.Engine)
	return server
}

func (s *OrderModuleTestSuite) TestHandler_PreviewOrder() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.PreviewOrderReq
		wantCode       int
		wantResp       test.Result[web.PreviewOrderResp]
	}{
		{
			name: "获取成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()
				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				mockPaymentSvc.EXPECT().GetPaymentChannels(gomock.Any()).Return([]payment.Channel{
					{Type: 1, Desc: "积分"},
					{Type: 2, Desc: "微信"},
				})

				pm := &payment.Module{Svc: mockPaymentSvc}

				mockProductSvc := productmocks.NewMockService(ctrl)
				mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).Return(product.SKU{
					ID:       100,
					SPUID:    100,
					SN:       "SKU100",
					Image:    "SKUImage100",
					Name:     "商品SKU100",
					Desc:     "商品SKU100",
					Price:    990,
					Stock:    10,
					SaleType: product.SaleTypeUnlimited, // 无限制
					Status:   product.StatusOnShelf,
				}, nil)
				ppm := &product.Module{Svc: mockProductSvc}

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().GetCreditsByUID(gomock.Any(), testUID).AnyTimes().Return(credit.Credit{
					TotalAmount: 1000,
				}, nil)
				cm := &credit.Module{Svc: mockCreditSvc}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.PreviewOrderReq{
				SKUs: []web.SKU{
					{
						SN:       "SKU100",
						Quantity: 1,
					},
				},
			},
			wantCode: 200,
			wantResp: test.Result[web.PreviewOrderResp]{
				Data: web.PreviewOrderResp{
					Order: web.Order{
						Payment: web.Payment{
							Items: []web.PaymentItem{
								{Type: int64(payment.ChannelTypeCredit)},
								{Type: int64(payment.ChannelTypeWechat)},
							},
						},
						OriginalTotalAmt: 990,
						RealTotalAmt:     990,
						Items: []web.OrderItem{
							{
								SKU: web.SKU{
									SN:            "SKU100",
									Image:         "SKUImage100",
									Name:          "商品SKU100",
									Desc:          "商品SKU100",
									OriginalPrice: 990,
									RealPrice:     990,
									Quantity:      1,
								},
							},
						},
					},
					Credits: 1000,
					Policy:  "请注意: 虚拟商品、一旦支付成功不退、不换,请谨慎操作",
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			req, err := http.NewRequest(http.MethodPost,
				"/order/preview", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.PreviewOrderResp]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_PreviewOrderFailed() {
	t := s.T()

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.PreviewOrderReq
		wantCode       int
		wantResp       test.Result[any]
	}{
		{
			name: "商品SKUSN不存在",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				pm := &payment.Module{Svc: paymentmocks.NewMockService(ctrl)}

				mockProductSvc := productmocks.NewMockService(ctrl)
				mockErr := fmt.Errorf("mock: SKU SN非法")
				mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).Return(product.SKU{}, mockErr)
				ppm := &product.Module{Svc: mockProductSvc}

				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.PreviewOrderReq{
				SKUs: []web.SKU{
					{
						SN:       "InvalidSKUSN",
						Quantity: 1,
					},
				},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "要购买的商品数量非法",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				pm := &payment.Module{Svc: mockPaymentSvc}

				mockProductSvc := productmocks.NewMockService(ctrl)
				mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).Return(product.SKU{
					ID:       100,
					SPUID:    100,
					SN:       "SKU100",
					Image:    "SKUImage100",
					Name:     "商品SKU100",
					Desc:     "商品SKU100",
					Price:    990,
					Stock:    10,
					SaleType: product.SaleTypeUnlimited, // 无限制
					Status:   product.StatusOnShelf,
				}, nil)
				ppm := &product.Module{Svc: mockProductSvc}

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				cm := &credit.Module{Svc: mockCreditSvc}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.PreviewOrderReq{
				SKUs: []web.SKU{
					{
						SN:       "SKU100",
						Quantity: 0,
					},
				},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "商品库存不足",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				pm := &payment.Module{Svc: mockPaymentSvc}

				mockProductSvc := productmocks.NewMockService(ctrl)
				mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).Return(product.SKU{
					ID:       100,
					SPUID:    100,
					SN:       "SKU100",
					Image:    "SKUImage100",
					Name:     "商品SKU100",
					Desc:     "商品SKU100",
					Price:    990,
					Stock:    10,
					SaleType: product.SaleTypeUnlimited, // 无限制
					Status:   product.StatusOnShelf,
				}, nil)
				ppm := &product.Module{Svc: mockProductSvc}

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				cm := &credit.Module{Svc: mockCreditSvc}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.PreviewOrderReq{
				SKUs: []web.SKU{
					{
						SN:       "SKU100",
						Quantity: 11,
					},
				},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "获取用户积分数失败",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				pm := &payment.Module{Svc: mockPaymentSvc}

				mockProductSvc := productmocks.NewMockService(ctrl)
				mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).Return(product.SKU{
					ID:       100,
					SPUID:    100,
					SN:       "SKU100",
					Image:    "SKUImage100",
					Name:     "商品SKU100",
					Desc:     "商品SKU100",
					Price:    990,
					Stock:    10,
					SaleType: product.SaleTypeUnlimited, // 无限制
					Status:   product.StatusOnShelf,
				}, nil)
				ppm := &product.Module{Svc: mockProductSvc}

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockErr := fmt.Errorf("mock: 获取积分失败")
				mockCreditSvc.EXPECT().GetCreditsByUID(gomock.Any(), testUID).AnyTimes().Return(credit.Credit{}, mockErr)
				cm := &credit.Module{Svc: mockCreditSvc}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.PreviewOrderReq{
				SKUs: []web.SKU{
					{
						SN:       "SKU100",
						Quantity: 10,
					},
				},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		// todo: 要购买商品超过库存限制(stockLimit)但是库存充足
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			req, err := http.NewRequest(http.MethodPost,
				"/order/preview", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_CreateOrder() {
	t := s.T()
	var testCases = []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.CreateOrderReq
		wantCode       int
		after          func(t *testing.T)
		assertRespFunc func(t *testing.T, resp test.Result[web.CreateOrderResp])
	}{
		{
			name: "创建成功_仅积分支付",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				id := int64(1)
				var pmt *payment.Payment
				mockPaymentSvc.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, p payment.Payment) (payment.Payment, error) {
					pmt = &payment.Payment{
						ID:               id,
						SN:               fmt.Sprintf("PaymentSN-create-order-%d", id),
						OrderID:          p.OrderID,
						OrderSN:          p.OrderSN,
						PayerID:          p.PayerID,
						OrderDescription: p.OrderDescription,
						TotalAmount:      p.TotalAmount,
						Records: []payment.Record{
							{
								PaymentNO3rd: "credit-1",
								Channel:      payment.ChannelTypeCredit,
								Amount:       990,
							},
						},
					}
					return *pmt, nil
				})

				mockPaymentSvc.EXPECT().PayByID(gomock.Any(), id).DoAndReturn(func(ctx context.Context, i int64) (payment.Payment, error) {
					pmt.Records[0].Status = payment.StatusPaidSuccess
					return *pmt, nil
				})

				pm := &payment.Module{Svc: mockPaymentSvc}

				mockProductSvc := productmocks.NewMockService(ctrl)
				mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).Return(product.SKU{
					ID:       100,
					SPUID:    100,
					SN:       "SKU100",
					Image:    "SKUImage100",
					Name:     "商品SKU100",
					Desc:     "商品SKU100",
					Price:    990,
					Stock:    10,
					SaleType: product.SaleTypeUnlimited, // 无限制
					Status:   product.StatusOnShelf,
				}, nil)
				ppm := &product.Module{Svc: mockProductSvc}

				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.CreateOrderReq{
				RequestID: "requestID01",
				SKUs: []web.SKU{
					{
						SN:       "SKU100",
						Quantity: 1,
					},
				},
				PaymentItems: []web.PaymentItem{
					{Type: int64(payment.ChannelTypeCredit), Amount: 990},
				},
			},
			wantCode: 200,
			assertRespFunc: func(t *testing.T, result test.Result[web.CreateOrderResp]) {
				t.Helper()
				assert.NotZero(t, result.Data.SN)
				assert.Zero(t, result.Data.WechatCodeURL)
			},
			after: func(t *testing.T) {
				t.Helper()
				orders, _, err := s.svc.FindUserVisibleOrdersByUID(context.Background(), testUID, 0, 1)
				require.NoError(t, err)
				require.Equal(t, domain.StatusProcessing, orders[0].Status)
			},
		},
		{
			name: "创建成功_积分和微信组合支付",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				id := int64(2)
				var pmt *payment.Payment
				mockPaymentSvc.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, p payment.Payment) (payment.Payment, error) {
					pmt = &payment.Payment{
						ID:               id,
						SN:               fmt.Sprintf("PaymentSN-create-order-%d", id),
						OrderID:          p.OrderID,
						OrderSN:          p.OrderSN,
						PayerID:          p.PayerID,
						OrderDescription: p.OrderDescription,
						TotalAmount:      p.TotalAmount,
						Records: []payment.Record{
							{
								PaymentNO3rd: "credit-1",
								Channel:      payment.ChannelTypeCredit,
								Amount:       1000,
							},
							{
								PaymentNO3rd: "wechat-2",
								Channel:      payment.ChannelTypeWechat,
								Amount:       8990,
							},
						},
					}
					return *pmt, nil
				})

				mockPaymentSvc.EXPECT().PayByID(gomock.Any(), id).DoAndReturn(func(ctx context.Context, i int64) (payment.Payment, error) {
					pmt.Records[0].Status = payment.StatusProcessing
					pmt.Records[1].Status = payment.StatusProcessing
					pmt.Records[1].WechatCodeURL = "webchat_code"
					return *pmt, nil
				})
				pm := &payment.Module{Svc: mockPaymentSvc}

				mockProductSvc := productmocks.NewMockService(ctrl)
				mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).Return(product.SKU{
					ID:       101,
					SPUID:    101,
					SN:       "SKU101",
					Image:    "SKUImage101",
					Name:     "商品SKU101",
					Desc:     "商品SKU101",
					Price:    9900,
					Stock:    1,
					SaleType: product.SaleTypeUnlimited, // 无限制
					Status:   product.StatusOnShelf,
				}, nil)
				ppm := &product.Module{Svc: mockProductSvc}

				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.CreateOrderReq{
				RequestID: "requestID02",
				SKUs: []web.SKU{
					{
						SN:       "SKU101",
						Quantity: 1,
					},
				},
				PaymentItems: []web.PaymentItem{
					{Type: int64(payment.ChannelTypeCredit), Amount: 5000},
					{Type: int64(payment.ChannelTypeWechat), Amount: 4900},
				},
			},
			wantCode: 200,
			assertRespFunc: func(t *testing.T, result test.Result[web.CreateOrderResp]) {
				t.Helper()
				assert.NotZero(t, result.Data.SN)
				assert.NotZero(t, result.Data.WechatCodeURL)
			},
			after: func(t *testing.T) {
				t.Helper()
				orders, _, err := s.svc.FindUserVisibleOrdersByUID(context.Background(), testUID, 1, 1)
				require.NoError(t, err)
				require.Equal(t, domain.StatusProcessing, orders[0].Status)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				_, err := s.cache.Delete(context.Background(), fmt.Sprintf("order:create:%s", tc.req.RequestID))
				require.NoError(t, err)
			})

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/order/create", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CreateOrderResp]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.assertRespFunc(t, recorder.MustScan())
			tc.after(t)
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_CreateOrderFailed() {
	t := s.T()
	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.CreateOrderReq
		wantCode       int
		wantResp       test.Result[any]
	}{
		{
			name: "请求ID为空",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				pm := &payment.Module{Svc: paymentmocks.NewMockService(ctrl)}
				ppm := &product.Module{Svc: productmocks.NewMockService(ctrl)}
				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.CreateOrderReq{
				RequestID: "",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "商品信息非法",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				pm := &payment.Module{Svc: paymentmocks.NewMockService(ctrl)}
				ppm := &product.Module{Svc: productmocks.NewMockService(ctrl)}
				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.CreateOrderReq{
				RequestID: "requestID01",
				SKUs:      []web.SKU{},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		// todo: SPU信息不存在
		{
			name: "商品SKUSN不存在",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				pm := &payment.Module{Svc: paymentmocks.NewMockService(ctrl)}

				mockProductSvc := productmocks.NewMockService(ctrl)
				mockErr := fmt.Errorf("mock: SKU SN非法")
				mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).Return(product.SKU{}, mockErr)
				ppm := &product.Module{Svc: mockProductSvc}

				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.CreateOrderReq{
				RequestID: "requestID02",
				SKUs: []web.SKU{
					{
						SN: "InvalidSKUSN",
					},
				},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "要购买的商品数量非法",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()
				return s.createOrderFailedHandler(t, ctrl)
			},
			req: web.CreateOrderReq{
				RequestID: "requestID03",
				SKUs: []web.SKU{
					{
						SN:       "SKU100",
						Quantity: 0,
					},
				},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "商品库存不足",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()
				return s.createOrderFailedHandler(t, ctrl)
			},
			req: web.CreateOrderReq{
				RequestID: "requestID04",
				SKUs: []web.SKU{
					{
						SN:       "SKU101",
						Quantity: 11,
					},
				},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "支付渠道非法",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()
				return s.createOrderFailedHandler(t, ctrl)
			},
			req: web.CreateOrderReq{
				RequestID: "requestID05",
				SKUs: []web.SKU{
					{
						SN:       "SKU101",
						Quantity: 1,
					},
				},
				PaymentItems: []web.PaymentItem{
					{
						Type: 0,
					},
				},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "商品总实价非法",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()
				return s.createOrderFailedHandler(t, ctrl)
			},
			req: web.CreateOrderReq{
				RequestID: "requestID06",
				SKUs: []web.SKU{
					{
						SN:       "SKU101",
						Quantity: 1,
					},
				},
				PaymentItems: []web.PaymentItem{
					{Type: int64(payment.ChannelTypeCredit), Amount: 3000},
					{Type: int64(payment.ChannelTypeWechat), Amount: 4900},
				},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "执行支付计划失败",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				id := int64(2)
				var pmt *payment.Payment
				mockPaymentSvc.EXPECT().CreatePayment(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, p payment.Payment) (payment.Payment, error) {
					pmt = &payment.Payment{
						ID:               id,
						SN:               fmt.Sprintf("PaymentSN-create-order-%d", id),
						OrderID:          p.OrderID,
						OrderSN:          p.OrderSN,
						PayerID:          p.PayerID,
						OrderDescription: p.OrderDescription,
						TotalAmount:      p.TotalAmount,
						Records: []payment.Record{
							{
								PaymentNO3rd: "credit-1",
								Channel:      payment.ChannelTypeCredit,
								Amount:       1000,
							},
							{
								PaymentNO3rd: "wechat-2",
								Channel:      payment.ChannelTypeWechat,
								Amount:       8990,
							},
						},
					}
					return *pmt, nil
				})

				mockErr := fmt.Errorf("mock: 支付ID非法")
				mockPaymentSvc.EXPECT().PayByID(gomock.Any(), id).Return(payment.Payment{}, mockErr)
				pm := &payment.Module{Svc: mockPaymentSvc}

				mockProductSvc := productmocks.NewMockService(ctrl)
				mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).Return(product.SKU{
					ID:       101,
					SPUID:    101,
					SN:       "SKU101",
					Image:    "SKUImage101",
					Name:     "商品SKU101",
					Desc:     "商品SKU101",
					Price:    9900,
					Stock:    1,
					SaleType: product.SaleTypeUnlimited, // 无限制
					Status:   product.StatusOnShelf,
				}, nil)
				ppm := &product.Module{Svc: mockProductSvc}

				mockCreditSvc := creditmocks.NewMockService(ctrl)
				mockCreditSvc.EXPECT().GetCreditsByUID(gomock.Any(), testUID).AnyTimes().Return(credit.Credit{
					TotalAmount: 10000,
				}, nil)
				cm := &credit.Module{Svc: mockCreditSvc}

				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.CreateOrderReq{
				RequestID: "requestID07",
				SKUs: []web.SKU{
					{
						SN:       "SKU101",
						Quantity: 1,
					},
				},
				PaymentItems: []web.PaymentItem{
					{Type: int64(payment.ChannelTypeCredit), Amount: 5000},
					{Type: int64(payment.ChannelTypeWechat), Amount: 4900},
				},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		// todo: 重复请求
		// todo: 要购买商品超过库存限制(stockLimit)但是库存充足
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				_, err := s.cache.Delete(context.Background(), fmt.Sprintf("order:create:%s", tc.req.RequestID))
				require.NoError(t, err)
			})

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/order/create", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			require.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) createOrderFailedHandler(t *testing.T, ctrl *gomock.Controller) *web.Handler {
	pm := &payment.Module{Svc: paymentmocks.NewMockService(ctrl)}

	mockProductSvc := productmocks.NewMockService(ctrl)
	mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).Return(product.SKU{
		ID:       101,
		SPUID:    101,
		SN:       "SKU101",
		Image:    "SKUImage101",
		Name:     "商品SKU101",
		Desc:     "商品SKU101",
		Price:    9900,
		Stock:    1,
		SaleType: product.SaleTypeUnlimited, // 无限制
		Status:   product.StatusOnShelf,
	}, nil)
	ppm := &product.Module{Svc: mockProductSvc}

	cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}

	handler, err := startup.InitHandler(pm, ppm, cm)
	require.NoError(t, err)
	return handler
}

func (s *OrderModuleTestSuite) TestHandler_Repay() {
	t := s.T()

	var testCases = []struct {
		name string

		before         func(t *testing.T)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.OrderSNReq
		wantCode       int
		assertRespFunc func(t *testing.T, result test.Result[web.RepayOrderResp])
	}{
		{
			name: "继续支付订单成功_返回二维码",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               11212,
					SN:               "orderSN-repay-11212",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(212),
					PaymentSn:        sqlx.NewNullString("paymentSN-repay-212"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusProcessing.ToUint8(),
				}, []dao.OrderItem{
					{
						OrderId:          11212,
						SPUId:            1,
						SKUId:            1,
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				id := int64(212)
				mockPaymentSvc.EXPECT().PayByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, pmtID int64) (payment.Payment, error) {
					return payment.Payment{
						ID:          id,
						SN:          fmt.Sprintf("paymentSN-repay-%d", id),
						OrderID:     11212,
						OrderSN:     fmt.Sprintf("orderSN-repay-11%d", id),
						TotalAmount: 9990,
						Records: []payment.Record{
							{
								PaymentNO3rd: "credit-1",
								Channel:      payment.ChannelTypeCredit,
								Amount:       1000,
								Status:       payment.StatusProcessing,
							},
							{
								PaymentNO3rd:  "wechat-2",
								Channel:       payment.ChannelTypeWechat,
								Amount:        8990,
								Status:        payment.StatusProcessing,
								WechatCodeURL: fmt.Sprintf("webchat_code-repay-%d", id),
							},
						},
					}, nil
				})
				pm := &payment.Module{Svc: mockPaymentSvc}
				ppm := &product.Module{Svc: productmocks.NewMockService(ctrl)}
				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}
				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler

			},
			req: web.OrderSNReq{
				SN: "orderSN-repay-11212",
			},
			wantCode: 200,
			assertRespFunc: func(t *testing.T, result test.Result[web.RepayOrderResp]) {
				t.Helper()
				assert.NotZero(t, result.Data.WechatCodeURL)
			},
		},
		{
			name: "继续支付订单成功_不返回二维码",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               11213,
					SN:               "orderSN-repay-11213",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(213),
					PaymentSn:        sqlx.NewNullString("paymentSN-repay-213"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusProcessing.ToUint8(),
				}, []dao.OrderItem{
					{
						OrderId:          11213,
						SPUId:            1,
						SKUId:            1,
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				id := int64(213)
				mockPaymentSvc.EXPECT().PayByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, pmtID int64) (payment.Payment, error) {
					return payment.Payment{
						ID:          id,
						SN:          fmt.Sprintf("paymentSN-repay-%d", id),
						OrderID:     11213,
						OrderSN:     fmt.Sprintf("orderSN-repay-11%d", id),
						TotalAmount: 9990,
						Records: []payment.Record{
							{
								PaymentNO3rd: "credit-1",
								Channel:      payment.ChannelTypeCredit,
								Amount:       9990,
								Status:       payment.StatusProcessing,
							},
						},
					}, nil
				})
				pm := &payment.Module{Svc: mockPaymentSvc}
				ppm := &product.Module{Svc: productmocks.NewMockService(ctrl)}
				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}
				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler

			},
			req: web.OrderSNReq{
				SN: "orderSN-repay-11213",
			},
			wantCode: 200,
			assertRespFunc: func(t *testing.T, result test.Result[web.RepayOrderResp]) {
				t.Helper()
				assert.Zero(t, result.Data.WechatCodeURL)
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/order/repay", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)

			recorder := test.NewJSONResponseRecorder[web.RepayOrderResp]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.assertRespFunc(t, recorder.MustScan())
		})
	}

}

func (s *OrderModuleTestSuite) TestHandler_RepayFailed() {
	t := s.T()

	testCases := []struct {
		name           string
		before         func(t *testing.T)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.OrderSNReq
		wantCode       int
		wantResp       test.Result[any]
	}{
		{
			name: "继续支付失败_订单状态非法_未支付",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               11111,
					SN:               "orderSN-repay-11",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(1000),
					PaymentSn:        sqlx.NewNullString("paymentSN-1000"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusInit.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-repay-11",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "继续支付失败_订单状态非法_支付失败",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               11112,
					SN:               "orderSN-repay-12",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(1001),
					PaymentSn:        sqlx.NewNullString("paymentSN-1001"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusFailed.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-repay-12",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "继续支付失败_订单状态非法_支付成功",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               11113,
					SN:               "orderSN-repay-13",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(1002),
					PaymentSn:        sqlx.NewNullString("paymentSN-1002"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusSuccess.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-repay-13",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "继续支付失败_订单状态非法_已取消",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               11114,
					SN:               "orderSN-repay-14",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(1003),
					PaymentSn:        sqlx.NewNullString("paymentSN-1003"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusCanceled.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-repay-14",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "继续支付失败_订单状态非法_超时关闭",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               11115,
					SN:               "orderSN-repay-15",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(1004),
					PaymentSn:        sqlx.NewNullString("paymentSN-1004"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusTimeoutClosed.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-repay-15",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "继续支付失败_无支付记录",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               11116,
					SN:               "orderSN-repay-16",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(1005),
					PaymentSn:        sqlx.NewNullString("paymentSN-1005"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusProcessing.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()
				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				mockErr := fmt.Errorf("mock: 无支付记录")
				mockPaymentSvc.EXPECT().PayByID(gomock.Any(), gomock.Any()).Return(payment.Payment{}, mockErr)
				pm := &payment.Module{Svc: mockPaymentSvc}
				ppm := &product.Module{Svc: productmocks.NewMockService(ctrl)}
				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}
				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.OrderSNReq{
				SN: "orderSN-repay-16",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name:           "订单序列号为空",
			before:         func(t *testing.T) {},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name:           "订单序列号非法",
			before:         func(t *testing.T) {},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "InvalidOrderSN",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/order/repay", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) emptyHandler(t *testing.T, ctrl *gomock.Controller) *web.Handler {
	t.Helper()
	pm := &payment.Module{Svc: paymentmocks.NewMockService(ctrl)}
	ppm := &product.Module{Svc: productmocks.NewMockService(ctrl)}
	cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}
	handler, err := startup.InitHandler(pm, ppm, cm)
	require.NoError(t, err)
	return handler
}

func (s *OrderModuleTestSuite) TestHandler_ListOrders() {
	t := s.T()

	s.TearDownTest()

	total := 100
	unpaidStatus := make([]uint8, 0, total)
	for idx := 0; idx < total; idx++ {
		id := int64(100 + idx)
		status := domain.OrderStatus(uint8(id)%6 + 1).ToUint8()
		if status == domain.StatusInit.ToUint8() {
			unpaidStatus = append(unpaidStatus, status)
		}
		orderEntity := dao.Order{
			Id:               id,
			SN:               fmt.Sprintf("OrderSN-list-%d", id),
			PaymentId:        sqlx.NewNullInt64(id),
			PaymentSn:        sqlx.NewNullString(fmt.Sprintf("PaymentSN-list-%d", id)),
			BuyerId:          testUID,
			OriginalTotalAmt: 100,
			RealTotalAmt:     100,
			Status:           status,
		}
		items := []dao.OrderItem{
			{
				SPUId:            id,
				SKUId:            id,
				SKUSN:            fmt.Sprintf("SKUSN-%d", id),
				SKUImage:         fmt.Sprintf("SKUImage-%d", id),
				SKUName:          fmt.Sprintf("SKUName-%d", id),
				SKUDescription:   fmt.Sprintf("SKUDescription-%d", id),
				SKUOriginalPrice: 100,
				SKURealPrice:     100,
				Quantity:         1,
			},
		}
		_, err := s.dao.CreateOrder(context.Background(), orderEntity, items)
		require.NoError(s.T(), err)
	}

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.ListOrdersReq

		wantCode int
		wantResp test.Result[web.ListOrdersResp]
	}{
		{
			name:           "获取成功",
			newHandlerFunc: s.emptyHandler,
			req: web.ListOrdersReq{
				Limit:  2,
				Offset: 0,
			},
			wantCode: 200,
			wantResp: test.Result[web.ListOrdersResp]{
				Data: web.ListOrdersResp{
					Total: int64(total - len(unpaidStatus)),
					Orders: []web.Order{
						{
							SN: "OrderSN-list-199",
							Payment: web.Payment{
								SN: fmt.Sprintf("PaymentSN-list-%d", 199),
							},
							OriginalTotalAmt: 100,
							RealTotalAmt:     100,
							Status:           domain.StatusProcessing.ToUint8(),
							Items: []web.OrderItem{
								{
									SKU: web.SKU{
										SN:            fmt.Sprintf("SKUSN-%d", 199),
										Image:         fmt.Sprintf("SKUImage-%d", 199),
										Name:          fmt.Sprintf("SKUName-%d", 199),
										Desc:          fmt.Sprintf("SKUDescription-%d", 199),
										OriginalPrice: 100,
										RealPrice:     100,
										Quantity:      1,
									},
								},
							},
						},
						{
							SN: "OrderSN-list-197",
							Payment: web.Payment{
								SN:    fmt.Sprintf("PaymentSN-list-%d", 197),
								Items: nil,
							},
							OriginalTotalAmt: 100,
							RealTotalAmt:     100,
							Status:           domain.StatusTimeoutClosed.ToUint8(),
							Items: []web.OrderItem{
								{
									SKU: web.SKU{
										SN:            fmt.Sprintf("SKUSN-%d", 197),
										Image:         fmt.Sprintf("SKUImage-%d", 197),
										Name:          fmt.Sprintf("SKUName-%d", 197),
										Desc:          fmt.Sprintf("SKUDescription-%d", 197),
										OriginalPrice: 100,
										RealPrice:     100,
										Quantity:      1,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/order/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.ListOrdersResp]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			s.assertListOrdersRespEqual(t, tc.wantResp.Data, recorder.MustScan().Data)
		})
	}
}

func (s *OrderModuleTestSuite) assertListOrdersRespEqual(t *testing.T, expected web.ListOrdersResp, actual web.ListOrdersResp) {
	assert.Equal(t, expected.Total, actual.Total)
	assert.Equal(t, len(expected.Orders), len(actual.Orders))
	for i := 0; i < len(actual.Orders); i++ {
		s.assertOrderEqual(t, expected.Orders[i], actual.Orders[i])
	}
}

func (s *OrderModuleTestSuite) assertOrderEqual(t *testing.T, expected web.Order, actual web.Order) {
	assert.NotZero(t, actual.Ctime)
	assert.NotZero(t, actual.Utime)
	actual.Ctime, actual.Utime = 0, 0
	assert.Equal(t, expected, actual)
}

func (s *OrderModuleTestSuite) TestHandler_RetrieveOrderDetail() {
	t := s.T()
	var testCases = []struct {
		name           string
		before         func(t *testing.T)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.OrderSNReq
		wantCode       int
		wantResp       test.Result[web.RetrieveOrderDetailResp]
	}{
		{
			name: "获取订单详情成功",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					SN:               "orderSN-33",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(33),
					PaymentSn:        sqlx.NewNullString("paymentSN-33"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusProcessing.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)

				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).Return(payment.Payment{
					ID:      33,
					SN:      "paymentSN-33",
					OrderID: 0,
					OrderSN: "orderSN-33",
					Records: []payment.Record{
						{
							Channel: payment.ChannelTypeCredit,
							Amount:  9900,
						},
					},
				}, nil)

				pm := &payment.Module{Svc: mockPaymentSvc}
				ppm := &product.Module{Svc: productmocks.NewMockService(ctrl)}
				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}
				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.OrderSNReq{
				SN: "orderSN-33",
			},
			wantCode: 200,
			wantResp: test.Result[web.RetrieveOrderDetailResp]{
				Data: web.RetrieveOrderDetailResp{
					Order: web.Order{
						SN: "orderSN-33",
						Payment: web.Payment{
							SN: "paymentSN-33",
							Items: []web.PaymentItem{
								{
									Type:   int64(payment.ChannelTypeCredit),
									Amount: 9900,
								},
							},
						},
						OriginalTotalAmt: 9900,
						RealTotalAmt:     9900,
						Status:           domain.StatusProcessing.ToUint8(),
						Items: []web.OrderItem{
							{
								SKU: web.SKU{
									SN:            fmt.Sprintf("SKUSN-%d", 1),
									Image:         fmt.Sprintf("SKUImage-%d", 1),
									Name:          "商品SKU",
									Desc:          "商品SKU描述",
									OriginalPrice: 9900,
									RealPrice:     9900,
									Quantity:      1,
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/order/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.RetrieveOrderDetailResp]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			s.assertOrderEqual(t, tc.wantResp.Data.Order, recorder.MustScan().Data.Order)
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_RetrieveOrderDetailFailed() {
	t := s.T()
	testCases := []struct {
		name           string
		before         func(t *testing.T)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.OrderSNReq
		wantCode       int
		wantResp       test.Result[any]
	}{
		{
			name: "订单状态非法_用户不可见状态",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					SN:               "orderSN-44",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(44),
					PaymentSn:        sqlx.NewNullString("paymentSN-44"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusInit.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-44",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name:           "订单序列号为空",
			before:         func(t *testing.T) {},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name:           "订单序列号非法",
			before:         func(t *testing.T) {},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "InvalidOrderSN",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "获取支付记录失败",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					SN:               "orderSN-55",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(55),
					PaymentSn:        sqlx.NewNullString("paymentSN-55"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusProcessing.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()

				mockPaymentSvc := paymentmocks.NewMockService(ctrl)
				mockErr := fmt.Errorf("mock: 获取支付记录失败")
				mockPaymentSvc.EXPECT().FindPaymentByID(gomock.Any(), gomock.Any()).Return(payment.Payment{}, mockErr)

				pm := &payment.Module{Svc: mockPaymentSvc}
				ppm := &product.Module{Svc: productmocks.NewMockService(ctrl)}
				cm := &credit.Module{Svc: creditmocks.NewMockService(ctrl)}
				handler, err := startup.InitHandler(pm, ppm, cm)
				require.NoError(t, err)
				return handler
			},
			req: web.OrderSNReq{
				SN: "orderSN-55",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/order/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_CancelOrder() {
	t := s.T()
	testCases := []struct {
		name string

		before         func(t *testing.T)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		after          func(t *testing.T)
		req            web.OrderSNReq
		wantCode       int
		wantResp       test.Result[any]
	}{
		{
			name: "取消订单成功",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					SN:        "orderSN-44",
					BuyerId:   testUID,
					PaymentId: sqlx.NewNullInt64(44),
					PaymentSn: sqlx.NewNullString("paymentSN-44"),
					Status:    domain.StatusProcessing.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			after: func(t *testing.T) {
				t.Helper()
				orderEntity, err := s.dao.FindOrderByUIDAndSNAndStatus(context.Background(), testUID, "orderSN-44", domain.StatusCanceled.ToUint8())
				assert.NoError(t, err)
				assert.NotZero(t, orderEntity)
			},
			req: web.OrderSNReq{
				SN: "orderSN-44",
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/order/cancel", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.RetrieveOrderDetailResp]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_CancelOrderFailed() {
	t := s.T()
	testCases := []struct {
		name           string
		before         func(t *testing.T)
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.OrderSNReq
		wantCode       int
		wantResp       test.Result[any]
	}{
		{
			name: "取消订单失败_订单状态非法_未支付",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               21111,
					SN:               "orderSN-cancel-11",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(2000),
					PaymentSn:        sqlx.NewNullString("paymentSN-2000"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusInit.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-cancel-11",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "取消订单失败_订单状态非法_支付失败",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               21112,
					SN:               "orderSN-cancel-12",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(2001),
					PaymentSn:        sqlx.NewNullString("paymentSN-2001"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusFailed.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-cancel-12",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "取消订单失败_订单状态非法_支付成功",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               21113,
					SN:               "orderSN-cancel-13",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(2002),
					PaymentSn:        sqlx.NewNullString("paymentSN-2002"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusSuccess.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-cancel-13",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "取消订单失败_订单状态非法_已取消",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               21114,
					SN:               "orderSN-cancel-14",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(2003),
					PaymentSn:        sqlx.NewNullString("paymentSN-2003"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusCanceled.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-cancel-14",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "取消订单失败_订单状态非法_超时关闭",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					Id:               21115,
					SN:               "orderSN-cancel-15",
					BuyerId:          testUID,
					PaymentId:        sqlx.NewNullInt64(2004),
					PaymentSn:        sqlx.NewNullString("paymentSN-2004"),
					OriginalTotalAmt: 9900,
					RealTotalAmt:     9900,
					Status:           domain.StatusTimeoutClosed.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUSN:            fmt.Sprintf("SKUSN-%d", 1),
						SKUImage:         fmt.Sprintf("SKUImage-%d", 1),
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)
			},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "orderSN-cancel-15",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name:           "订单序列号为空",
			before:         func(t *testing.T) {},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name:           "订单序列号非法",
			before:         func(t *testing.T) {},
			newHandlerFunc: s.emptyHandler,
			req: web.OrderSNReq{
				SN: "InvalidOrderSN",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/order/cancel", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestPaymentConsumer_Consume() {
	t := s.T()

	producer, er := s.mq.Producer("payment_events")
	require.NoError(t, er)

	testCases := []struct {
		name string

		before         func(t *testing.T, producer mq.Producer, message *mq.Message)
		evt            event.PaymentEvent
		after          func(t *testing.T, orderSN string)
		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name: "设置支付成功成功",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					SN:        "orderSN-PaymentConsumer-22",
					BuyerId:   testUID,
					PaymentId: sqlx.NewNullInt64(22),
					PaymentSn: sqlx.NewNullString("paymentSN-22"),
					Status:    domain.StatusProcessing.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)

				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)

				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			evt: event.PaymentEvent{
				OrderSN: "orderSN-PaymentConsumer-22",
				PayerID: testUID,
				Status:  uint8(payment.StatusPaidSuccess),
			},
			after: func(t *testing.T, orderSN string) {
				t.Helper()
				orderEntity, err := s.dao.FindOrderByUIDAndSNAndStatus(context.Background(), testUID, orderSN, domain.StatusSuccess.ToUint8())
				assert.NoError(t, err)
				assert.Equal(t, domain.StatusSuccess.ToUint8(), orderEntity.Status)
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "设置支付成功_忽略订单序列号为空",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			evt: event.PaymentEvent{
				OrderSN: "",
				PayerID: testUID,
				Status:  uint8(payment.StatusPaidSuccess),
			},
			after: func(t *testing.T, orderSN string) {
				t.Helper()
				_, err := s.dao.FindOrderByUIDAndSNAndStatus(context.Background(), testUID, orderSN, domain.StatusSuccess.ToUint8())
				assert.Error(t, err)
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "设置支付成功_忽略订单序列号非法",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			evt: event.PaymentEvent{
				OrderSN: "InvalidOrderSN",
				PayerID: testUID,
				Status:  uint8(payment.StatusPaidSuccess),
			},
			after: func(t *testing.T, orderSN string) {
				t.Helper()
				_, err := s.dao.FindOrderByUIDAndSNAndStatus(context.Background(), testUID, orderSN, domain.StatusSuccess.ToUint8())
				assert.Error(t, err)
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "设置支付成功失败_买家ID非法",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			evt: event.PaymentEvent{
				OrderSN: "OrderSN-3",
				PayerID: 0,
				Status:  uint8(payment.StatusPaidSuccess),
			},
			after: func(t *testing.T, orderSN string) {
				t.Helper()
				_, err := s.dao.FindOrderByUIDAndSNAndStatus(context.Background(), 0, orderSN, domain.StatusSuccess.ToUint8())
				assert.Error(t, err)
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "设置支付失败成功",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					SN:        "orderSN-23",
					BuyerId:   testUID,
					PaymentId: sqlx.NewNullInt64(23),
					PaymentSn: sqlx.NewNullString("paymentSN-23"),
					Status:    domain.StatusProcessing.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)

				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)

				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			evt: event.PaymentEvent{
				OrderSN: "orderSN-23",
				PayerID: testUID,
				Status:  uint8(payment.StatusPaidFailed),
			},
			after: func(t *testing.T, orderSN string) {
				t.Helper()
				orderEntity, err := s.dao.FindOrderByUIDAndSNAndStatus(context.Background(), testUID, orderSN, domain.StatusFailed.ToUint8())
				assert.NoError(t, err)
				assert.Equal(t, domain.StatusFailed.ToUint8(), orderEntity.Status)
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "设置支付失败或成功失败_支付状态非法",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					SN:        "orderSN-24",
					BuyerId:   testUID,
					PaymentId: sqlx.NewNullInt64(24),
					PaymentSn: sqlx.NewNullString("paymentSN-24"),
					Status:    domain.StatusProcessing.ToUint8(),
				}, []dao.OrderItem{
					{
						SPUId:            1,
						SKUId:            1,
						SKUName:          "商品SKU",
						SKUDescription:   "商品SKU描述",
						SKUOriginalPrice: 9900,
						SKURealPrice:     9900,
						Quantity:         1,
					},
				})
				require.NoError(t, err)

				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)

				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			evt: event.PaymentEvent{
				OrderSN: "OrderSN-24",
				PayerID: testUID,
			},
			after: func(t *testing.T, orderSN string) {
				t.Helper()
				orderEntity, err := s.dao.FindOrderByUIDAndSNAndStatus(context.Background(), testUID, orderSN, domain.StatusProcessing.ToUint8())
				assert.NoError(t, err)
				assert.Equal(t, domain.StatusProcessing.ToUint8(), orderEntity.Status)
			},
			errRequireFunc: require.Error,
		},
	}

	consumer, err := event.NewPaymentConsumer(s.svc, s.mq)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message := s.newPaymentEvent(t, tc.evt)
			tc.before(t, producer, message)

			err = consumer.Consume(context.Background())
			tc.errRequireFunc(t, err)

			err = consumer.Consume(context.Background())
			tc.errRequireFunc(t, err)

			tc.after(t, tc.evt.OrderSN)
		})
	}
}

func (s *OrderModuleTestSuite) newPaymentEvent(t *testing.T, evt event.PaymentEvent) *mq.Message {
	t.Helper()
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *OrderModuleTestSuite) TestJob_CloseTimeoutOrders() {
	t := s.T()

	total := 15

	testCases := []struct {
		name       string
		before     func(t *testing.T)
		getJobFunc func(t *testing.T) *job.CloseTimeoutOrdersJob
		after      func(t *testing.T)
	}{
		{
			name: "关闭超时订单成功_正常情况",
			before: func(t *testing.T) {
				t.Helper()
				for idx := 0; idx < total; idx++ {
					id := int64(200 + idx)
					orderEntity := dao.Order{
						Id:               id,
						SN:               fmt.Sprintf("OrderSN-close-%d", id),
						PaymentId:        sqlx.NewNullInt64(id),
						PaymentSn:        sqlx.NewNullString(fmt.Sprintf("PaymentSN-close-%d", id)),
						BuyerId:          id,
						OriginalTotalAmt: 100,
						RealTotalAmt:     100,
					}
					items := []dao.OrderItem{
						{
							SPUId:            id,
							SKUId:            id,
							SKUName:          fmt.Sprintf("SKUName-%d", id),
							SKUDescription:   fmt.Sprintf("SKUDescription-%d", id),
							SKUOriginalPrice: 100,
							SKURealPrice:     100,
							Quantity:         1,
						},
					}
					_, err := s.dao.CreateOrder(context.Background(), orderEntity, items)
					require.NoError(s.T(), err)
				}
			},
			getJobFunc: func(t *testing.T) *job.CloseTimeoutOrdersJob {
				t.Helper()
				return job.NewCloseTimeoutOrdersJob(s.svc, 0, 0, 10)
			},
			after: func(t *testing.T) {
				t.Helper()
				for idx := 0; idx < total; idx++ {
					id := int64(200 + idx)
					orderEntity, err := s.dao.FindOrderByUIDAndSNAndStatus(context.Background(), id, fmt.Sprintf("OrderSN-close-%d", id),
						domain.StatusTimeoutClosed.ToUint8())
					assert.NoError(t, err)
					assert.Equal(t, domain.StatusTimeoutClosed.ToUint8(), orderEntity.Status)
				}
			},
		},
		{
			name: "关闭超时订单成功_边界情况",
			before: func(t *testing.T) {
				t.Helper()
				for idx := 0; idx < total; idx++ {
					id := int64(300 + idx)
					orderEntity := dao.Order{
						Id:               id,
						SN:               fmt.Sprintf("OrderSN-close-%d", id),
						PaymentId:        sqlx.NewNullInt64(id),
						PaymentSn:        sqlx.NewNullString(fmt.Sprintf("PaymentSN-close-%d", id)),
						BuyerId:          id,
						OriginalTotalAmt: 100,
						RealTotalAmt:     100,
					}
					items := []dao.OrderItem{
						{
							SPUId:            id,
							SKUId:            id,
							SKUName:          fmt.Sprintf("SKUName-%d", id),
							SKUDescription:   fmt.Sprintf("SKUDescription-%d", id),
							SKUOriginalPrice: 100,
							SKURealPrice:     100,
							Quantity:         1,
						},
					}
					_, err := s.dao.CreateOrder(context.Background(), orderEntity, items)
					require.NoError(s.T(), err)
				}
			},
			getJobFunc: func(t *testing.T) *job.CloseTimeoutOrdersJob {
				t.Helper()
				return job.NewCloseTimeoutOrdersJob(s.svc, 0, 0, total)
			},
			after: func(t *testing.T) {
				t.Helper()
				for idx := 0; idx < total; idx++ {
					id := int64(300 + idx)
					orderEntity, err := s.dao.FindOrderByUIDAndSNAndStatus(context.Background(), id, fmt.Sprintf("OrderSN-close-%d", id),
						domain.StatusTimeoutClosed.ToUint8())
					assert.NoError(t, err)
					assert.Equal(t, domain.StatusTimeoutClosed.ToUint8(), orderEntity.Status)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			j := tc.getJobFunc(t)
			require.NotZero(t, j.Name())
			require.NoError(t, j.Run(context.Background()))
			tc.after(t)
		})
	}
}
