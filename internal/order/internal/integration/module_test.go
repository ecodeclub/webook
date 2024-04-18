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
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
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

type fakePaymentService struct {
	counter atomic.Int64
}

func (f *fakePaymentService) CreatePayment(_ context.Context, p payment.Payment) (payment.Payment, error) {
	f.counter.Add(1)
	id := f.counter.Load()

	columns := map[int64]payment.Payment{
		1: {
			ID:          1,
			SN:          "PaymentSN-1",
			OrderID:     p.OrderID,
			OrderSN:     p.OrderSN,
			TotalAmount: p.TotalAmount,
			PayDDL:      p.PayDDL,
			Records: []payment.Record{
				{
					PaymentNO3rd: "credit-1",
					Channel:      payment.ChannelTypeCredit,
					Amount:       990,
					Status:       0,
				},
			},
		},
		2: {
			ID:          2,
			SN:          "PaymentSN-2",
			OrderID:     p.OrderID,
			OrderSN:     p.OrderSN,
			TotalAmount: p.TotalAmount,
			PayDDL:      p.PayDDL,
			Records: []payment.Record{
				{
					PaymentNO3rd: "credit-1",
					Channel:      payment.ChannelTypeCredit,
					Amount:       1000,
					Status:       0,
				},
				{
					PaymentNO3rd:  "wechat-2",
					Channel:       payment.ChannelTypeWechat,
					Amount:        8990,
					Status:        0,
					WechatCodeURL: "webchat_code",
				},
			},
		},
	}
	r, ok := columns[id]
	if !ok {
		return payment.Payment{}, fmt.Errorf(fmt.Sprintf("未配置的支付id=%d", id))
	}
	return r, nil
}

func (f *fakePaymentService) GetPaymentChannels(_ context.Context) []payment.Channel {
	return []payment.Channel{
		{Type: 1, Desc: "积分"},
		{Type: 2, Desc: "微信"},
	}
}

func (f *fakePaymentService) FindPaymentByID(_ context.Context, paymentID int64) (payment.Payment, error) {
	payments := map[int64]payment.Payment{
		33: {
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
		},
	}
	p, ok := payments[paymentID]
	if !ok {
		return payment.Payment{}, fmt.Errorf(fmt.Sprintf("未配置的支付ID = %d", paymentID))
	}
	return p, nil
}

func TestOrderModule(t *testing.T) {
	suite.Run(t, new(OrderModuleTestSuite))
}

type OrderModuleTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	mq     mq.MQ
	dao    dao.OrderDAO
	cache  ecache.Cache
	svc    order.Service
	ctrl   *gomock.Controller
}

func (s *OrderModuleTestSuite) SetupSuite() {

	s.ctrl = gomock.NewController(s.T())

	handler, err := startup.InitHandler(&fakePaymentService{}, s.getProductMockService(), s.getCreditMockService())
	require.NoError(s.T(), err)

	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: testUID,
		}))
	})
	handler.PrivateRoutes(server.Engine)

	s.server = server
	s.db = testioc.InitDB()
	err = dao.InitTables(s.db)
	require.NoError(s.T(), err)
	s.dao = dao.NewOrderGORMDAO(s.db)
	s.svc = order.InitService(s.db)
	s.mq = testioc.InitMQ()
	s.cache = testioc.InitCache()
}

func (s *OrderModuleTestSuite) getProductMockService() *productmocks.MockService {
	mockedProductSvc := productmocks.NewMockService(s.ctrl)
	skus := map[string]product.SKU{
		"SKU100": {
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
		},
		"SKU101": {
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
		},
	}
	spus := map[int64]product.SPU{
		100: {
			ID:   100,
			SN:   "SPU100",
			Name: "商品SPU100",
			Desc: "商品SKU100",
			SKUs: []product.SKU{
				skus["SKU100"],
			},
			Status: product.StatusOnShelf,
		},
		101: {
			ID:   101,
			SN:   "SPU101",
			Name: "商品SPU101",
			Desc: "商品SKU101",
			SKUs: []product.SKU{
				skus["SKU101"],
			},
			Status: product.StatusOnShelf,
		},
	}
	mockedProductSvc.EXPECT().FindSKUBySN(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, sn string) (product.SKU, error) {
		sku, ok := skus[sn]
		if !ok {
			return product.SKU{}, errors.New("SKU的SN非法")
		}
		return sku, nil
	}).AnyTimes()
	mockedProductSvc.EXPECT().FindSPUByID(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, id int64) (product.SPU, error) {
		spu, ok := spus[id]
		if !ok {
			return product.SPU{}, errors.New("SPU的ID非法")
		}
		return spu, nil
	}).AnyTimes()

	return mockedProductSvc
}

func (s *OrderModuleTestSuite) getCreditMockService() *creditmocks.MockService {
	mockedCreditSvc := creditmocks.NewMockService(s.ctrl)
	mockedCreditSvc.EXPECT().GetCreditsByUID(gomock.Any(), testUID).AnyTimes().Return(credit.Credit{
		TotalAmount: 1000,
	}, nil)
	return mockedCreditSvc
}

func (s *OrderModuleTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `orders`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("DROP TABLE `order_items`").Error
	require.NoError(s.T(), err)

	s.ctrl.Finish()
}

func (s *OrderModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `orders`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `order_items`").Error
	require.NoError(s.T(), err)
}

func (s *OrderModuleTestSuite) TestHandler_PreviewOrder() {

	testCases := []struct {
		name string

		req      web.PreviewOrderReq
		wantCode int
		wantResp test.Result[web.PreviewOrderResp]
	}{
		{
			name: "获取成功",
			req: web.PreviewOrderReq{
				SKUSN:    "SKU100",
				Quantity: 1,
			},
			wantCode: 200,
			wantResp: test.Result[web.PreviewOrderResp]{
				Data: web.PreviewOrderResp{
					Credits: 1000,
					Payments: []web.PaymentItem{
						{Type: payment.ChannelTypeCredit},
						{Type: payment.ChannelTypeWechat},
					},
					SKUs: []web.SKU{
						{
							SN:            "SKU100",
							Image:         "SKUImage100",
							Name:          "商品SKU100",
							Desc:          "商品SKU100",
							OriginalPrice: 990,
							RealPrice:     990,
							Quantity:      1,
						},
					},
					Policy: "请注意: 虚拟商品、一旦支持成功不退、不换,请谨慎操作",
				},
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/order/preview", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.PreviewOrderResp]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_PreviewOrderFailed() {
	testCases := []struct {
		name string

		req      web.PreviewOrderReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "商品SKUSN不存在",
			req: web.PreviewOrderReq{
				SKUSN:    "InvalidSKUSN",
				Quantity: 1,
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "要购买的商品数量非法",
			req: web.PreviewOrderReq{
				SKUSN:    "SKU100",
				Quantity: 0,
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "商品库存不足",
			req: web.PreviewOrderReq{
				SKUSN:    "SKU100",
				Quantity: 11,
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
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/order/preview", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_CreateOrderAndPayment() {
	var testCases = []struct {
		name           string
		req            web.CreateOrderReq
		wantCode       int
		assertRespFunc func(t *testing.T, resp test.Result[web.CreateOrderResp])
	}{
		{
			name: "创建成功_仅积分支付",
			req: web.CreateOrderReq{
				RequestID: "requestID01",
				SKUs: []web.SKU{
					{
						SN:       "SKU100",
						Quantity: 1,
					},
				},
				PaymentItems: []web.PaymentItem{
					{Type: payment.ChannelTypeCredit, Amount: 990},
				},
			},
			wantCode: 200,
			assertRespFunc: func(t *testing.T, result test.Result[web.CreateOrderResp]) {
				t.Helper()
				assert.NotZero(t, result.Data.OrderSN)
				assert.Zero(t, result.Data.WechatCodeURL)
			},
		},
		// todo: 创建成功_仅微信支付
		{
			name: "创建成功_积分和微信组合支付",
			req: web.CreateOrderReq{
				RequestID: "requestID02",
				SKUs: []web.SKU{
					{
						SN:       "SKU101",
						Quantity: 1,
					},
				},
				PaymentItems: []web.PaymentItem{
					{Type: payment.ChannelTypeCredit, Amount: 5000},
					{Type: payment.ChannelTypeWechat, Amount: 4900},
				},
			},
			wantCode: 200,
			assertRespFunc: func(t *testing.T, result test.Result[web.CreateOrderResp]) {
				t.Helper()
				assert.NotZero(t, result.Data.OrderSN)
				assert.NotZero(t, result.Data.WechatCodeURL)
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/order/create", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.CreateOrderResp]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.assertRespFunc(t, recorder.MustScan())

			_, err = s.cache.Delete(context.Background(), fmt.Sprintf("order:create:%s", tc.req.RequestID))
			require.NoError(t, err)
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_CreateOrderAndPaymentFailed() {
	testCases := []struct {
		name string

		req      web.CreateOrderReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "请求ID为空",
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
			req: web.CreateOrderReq{
				RequestID: "requestID09",
				SKUs:      []web.SKU{},
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "商品SKUSN不存在",
			req: web.CreateOrderReq{
				RequestID: "requestID03",
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
			req: web.CreateOrderReq{
				RequestID: "requestID04",
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
			req: web.CreateOrderReq{
				RequestID: "requestID05",
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
			name: "商品总原价非法",
			req: web.CreateOrderReq{
				RequestID: "requestID06",
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
		{
			name: "商品总实价非法",
			req: web.CreateOrderReq{
				RequestID: "requestID07",
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
		{
			name: "支付渠道非法",
			req: web.CreateOrderReq{
				RequestID: "requestID08",
				SKUs: []web.SKU{
					{
						SN:       "SKU100",
						Quantity: 10,
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
			name: "请求重复",
			req: web.CreateOrderReq{
				RequestID: "requestID08",
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
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/order/create", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_RetrieveOrderStatus() {
	var testCases = []struct {
		name string

		before         func(t *testing.T)
		req            web.RetrieveOrderStatusReq
		wantCode       int
		assertRespFunc func(t *testing.T, result test.Result[web.RetrieveOrderStatusResp])
	}{
		{
			name: "获取订单状态成功",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					SN:        "orderSN-1",
					BuyerId:   testUID,
					PaymentId: sqlx.NewNullInt64(12),
					PaymentSn: sqlx.NewNullString("paymentSN-12"),
				}, []dao.OrderItem{
					{
						Id:               0,
						OrderId:          0,
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

			req: web.RetrieveOrderStatusReq{
				OrderSN: "orderSN-1",
			},
			wantCode: 200,
			assertRespFunc: func(t *testing.T, result test.Result[web.RetrieveOrderStatusResp]) {
				t.Helper()
				assert.NotZero(t, result.Data.OrderStatus)
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/order", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.RetrieveOrderStatusResp]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.assertRespFunc(t, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_RetrieveOrderStatusFailed() {
	testCases := []struct {
		name     string
		req      web.RetrieveOrderStatusReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "订单序列号为空",
			req: web.RetrieveOrderStatusReq{
				OrderSN: "",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "订单序列号非法",
			req: web.RetrieveOrderStatusReq{
				OrderSN: "InvalidOrderSN",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/order", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_ListOrders() {

	total := 100
	for idx := 0; idx < total; idx++ {
		id := int64(100 + idx)
		orderEntity := dao.Order{
			Id:                 id,
			SN:                 fmt.Sprintf("OrderSN-list-%d", id),
			PaymentId:          sqlx.NewNullInt64(id),
			PaymentSn:          sqlx.NewNullString(fmt.Sprintf("PaymentSN-list-%d", id)),
			BuyerId:            testUID,
			OriginalTotalPrice: 100,
			RealTotalPrice:     100,
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
		name string
		req  web.ListOrdersReq

		wantCode int
		wantResp test.Result[web.ListOrdersResp]
	}{
		{
			name: "获取成功",
			req: web.ListOrdersReq{
				Limit:  2,
				Offset: 0,
			},
			wantCode: 200,
			wantResp: test.Result[web.ListOrdersResp]{
				Data: web.ListOrdersResp{
					Total: int64(total),
					Orders: []web.Order{
						{
							SN: "OrderSN-list-199",
							Payment: web.Payment{
								SN: fmt.Sprintf("PaymentSN-list-%d", 199),
							},
							OriginalTotalPrice: 100,
							RealTotalPrice:     100,
							Status:             domain.StatusUnpaid.ToUint8(),
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
							SN: "OrderSN-list-198",
							Payment: web.Payment{
								SN:    fmt.Sprintf("PaymentSN-list-%d", 198),
								Items: nil,
							},
							OriginalTotalPrice: 100,
							RealTotalPrice:     100,
							Status:             domain.StatusUnpaid.ToUint8(),
							Items: []web.OrderItem{
								{
									SKU: web.SKU{
										SN:            fmt.Sprintf("SKUSN-%d", 198),
										Image:         fmt.Sprintf("SKUImage-%d", 198),
										Name:          fmt.Sprintf("SKUName-%d", 198),
										Desc:          fmt.Sprintf("SKUDescription-%d", 198),
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
		{
			name: "获取部分",
			req: web.ListOrdersReq{
				Limit:  2,
				Offset: 99,
			},
			wantCode: 200,
			wantResp: test.Result[web.ListOrdersResp]{
				Data: web.ListOrdersResp{
					Total: int64(total),
					Orders: []web.Order{
						{
							SN: "OrderSN-list-100",
							Payment: web.Payment{
								SN: fmt.Sprintf("PaymentSN-list-%d", 100),
							},
							OriginalTotalPrice: 100,
							RealTotalPrice:     100,
							Status:             domain.StatusUnpaid.ToUint8(),
							Items: []web.OrderItem{
								{
									SKU: web.SKU{
										SN:            fmt.Sprintf("SKUSN-%d", 100),
										Image:         fmt.Sprintf("SKUImage-%d", 100),
										Name:          fmt.Sprintf("SKUName-%d", 100),
										Desc:          fmt.Sprintf("SKUDescription-%d", 100),
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
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/order/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.ListOrdersResp]()
			s.server.ServeHTTP(recorder, req)
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
	var testCases = []struct {
		name string

		before   func(t *testing.T)
		req      web.RetrieveOrderDetailReq
		wantCode int
		wantResp test.Result[web.RetrieveOrderDetailResp]
	}{
		{
			name: "获取订单详情成功",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					SN:                 "orderSN-33",
					BuyerId:            testUID,
					PaymentId:          sqlx.NewNullInt64(33),
					PaymentSn:          sqlx.NewNullString("paymentSN-33"),
					OriginalTotalPrice: 9900,
					RealTotalPrice:     9900,
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

			req: web.RetrieveOrderDetailReq{
				OrderSN: "orderSN-33",
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
									Type:   payment.ChannelTypeCredit,
									Amount: 9900,
								},
							},
						},
						OriginalTotalPrice: 9900,
						RealTotalPrice:     9900,
						Status:             domain.StatusUnpaid.ToUint8(),
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
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/order/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.RetrieveOrderDetailResp]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			s.assertOrderEqual(t, tc.wantResp.Data.Order, recorder.MustScan().Data.Order)
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_RetrieveOrderDetailFailed() {
	testCases := []struct {
		name     string
		req      web.RetrieveOrderDetailReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "订单序列号为空",
			req: web.RetrieveOrderDetailReq{
				OrderSN: "",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "订单序列号非法",
			req: web.RetrieveOrderDetailReq{
				OrderSN: "InvalidOrderSN",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/order/detail", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_CancelOrder() {
	var (
		testCases = []struct {
			name string

			before   func(t *testing.T)
			after    func(t *testing.T)
			req      web.CancelOrderReq
			wantCode int
			wantResp test.Result[any]
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
				after: func(t *testing.T) {
					t.Helper()
					orderEntity, err := s.dao.FindOrderByUIDAndSN(context.Background(), testUID, "orderSN-44")
					assert.NoError(t, err)
					assert.Equal(t, domain.StatusCanceled.ToUint8(), orderEntity.Status)
				},
				req: web.CancelOrderReq{
					OrderSN: "orderSN-44",
				},
				wantCode: 200,
				wantResp: test.Result[any]{
					Msg: "OK",
				},
			},
		}
	)
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/order/cancel", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.RetrieveOrderDetailResp]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			tc.after(t)
		})
	}
}

func (s *OrderModuleTestSuite) TestHandler_CancelOrderFailed() {
	testCases := []struct {
		name     string
		req      web.CancelOrderReq
		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "订单序列号为空",
			req: web.CancelOrderReq{
				OrderSN: "",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
		{
			name: "订单序列号非法",
			req: web.CancelOrderReq{
				OrderSN: "InvalidOrderSN",
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: errs.SystemError.Code,
				Msg:  errs.SystemError.Msg,
			},
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/order/cancel", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func (s *OrderModuleTestSuite) TestConsumer_ConsumeCompleteOrder() {
	t := s.T()

	producer, er := s.mq.Producer("order_complete_events")
	require.NoError(t, er)

	var testCases = []struct {
		name string

		before         func(t *testing.T, producer mq.Producer, message *mq.Message)
		evt            event.CompleteOrderEvent
		after          func(t *testing.T)
		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name: "完成订单成功",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := s.dao.CreateOrder(context.Background(), dao.Order{
					SN:        "orderSN-22",
					BuyerId:   testUID,
					PaymentId: sqlx.NewNullInt64(22),
					PaymentSn: sqlx.NewNullString("paymentSN-22"),
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
			evt: event.CompleteOrderEvent{
				OrderSN: "orderSN-22",
				BuyerID: testUID,
			},
			after: func(t *testing.T) {
				t.Helper()
				orderEntity, err := s.dao.FindOrderByUIDAndSN(context.Background(), testUID, "orderSN-22")
				assert.NoError(t, err)
				assert.Equal(t, domain.StatusCompleted.ToUint8(), orderEntity.Status)
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "完成订单失败_订单序列号为空",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			evt: event.CompleteOrderEvent{
				OrderSN: "",
				BuyerID: testUID,
			},
			after:          func(t *testing.T) {},
			errRequireFunc: require.Error,
		},
		{
			name: "完成订单失败_订单序列号非法",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			evt: event.CompleteOrderEvent{
				OrderSN: "InvalidOrderSN",
				BuyerID: testUID,
			},
			after:          func(t *testing.T) {},
			errRequireFunc: require.Error,
		},
		{
			name: "完成订单失败_买家ID非法",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			evt: event.CompleteOrderEvent{
				OrderSN: "OrderSN-3",
				BuyerID: 0,
			},
			after:          func(t *testing.T) {},
			errRequireFunc: require.Error,
		},
	}

	consumer, err := event.NewCompleteOrderConsumer(s.svc, s.mq)
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			message := s.newOrderCompleteEvent(t, tc.evt)
			tc.before(t, producer, message)

			err = consumer.Consume(context.Background())
			tc.errRequireFunc(t, err)

			err = consumer.Consume(context.Background())
			tc.errRequireFunc(t, err)

			tc.after(t)
		})
	}
}

func (s *OrderModuleTestSuite) newOrderCompleteEvent(t *testing.T, evt event.CompleteOrderEvent) *mq.Message {
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
		getJobFunc func(t *testing.T) *job.CloseExpiredOrdersJob
		after      func(t *testing.T)
	}{
		{
			name: "关闭超时订单成功_正常情况",
			before: func(t *testing.T) {
				t.Helper()
				for idx := 0; idx < total; idx++ {
					id := int64(200 + idx)
					orderEntity := dao.Order{
						Id:                 id,
						SN:                 fmt.Sprintf("OrderSN-close-%d", id),
						PaymentId:          sqlx.NewNullInt64(id),
						PaymentSn:          sqlx.NewNullString(fmt.Sprintf("PaymentSN-close-%d", id)),
						BuyerId:            id,
						OriginalTotalPrice: 100,
						RealTotalPrice:     100,
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
			getJobFunc: func(t *testing.T) *job.CloseExpiredOrdersJob {
				t.Helper()
				return job.NewCloseExpiredOrdersJob(s.svc, 0, 0, 10)
			},
			after: func(t *testing.T) {
				t.Helper()
				for idx := 0; idx < total; idx++ {
					id := int64(200 + idx)
					orderEntity, err := s.dao.FindOrderBySN(context.Background(), fmt.Sprintf("OrderSN-close-%d", id))
					assert.NoError(t, err)
					assert.Equal(t, domain.StatusExpired.ToUint8(), orderEntity.Status)
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
						Id:                 id,
						SN:                 fmt.Sprintf("OrderSN-close-%d", id),
						PaymentId:          sqlx.NewNullInt64(id),
						PaymentSn:          sqlx.NewNullString(fmt.Sprintf("PaymentSN-close-%d", id)),
						BuyerId:            id,
						OriginalTotalPrice: 100,
						RealTotalPrice:     100,
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
			getJobFunc: func(t *testing.T) *job.CloseExpiredOrdersJob {
				t.Helper()
				return job.NewCloseExpiredOrdersJob(s.svc, 0, 0, total)
			},
			after: func(t *testing.T) {
				t.Helper()
				for idx := 0; idx < total; idx++ {
					id := int64(300 + idx)
					orderEntity, err := s.dao.FindOrderBySN(context.Background(), fmt.Sprintf("OrderSN-close-%d", id))
					assert.NoError(t, err)
					assert.Equal(t, domain.StatusExpired.ToUint8(), orderEntity.Status)
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
