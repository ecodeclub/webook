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
	"time"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/consumer"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/marketing/internal/service"
	"github.com/ecodeclub/webook/internal/marketing/internal/web"
	"github.com/ecodeclub/webook/internal/order"
	ordermocks "github.com/ecodeclub/webook/internal/order/mocks"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/test/mocks"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const testID = 275892

func TestMarketingModule(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

type ModuleTestSuite struct {
	suite.Suite
	db   *egorm.Component
	dao  dao.MarketingDAO
	repo repository.MarketingRepository
}

func (s *ModuleTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	s.NoError(dao.InitTables(s.db))
	s.dao = dao.NewGORMMarketingDAO(s.db)
	s.repo = repository.NewRepository(s.dao)
}

func (s *ModuleTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `redemption_codes`").Error
	s.NoError(err)
	err = s.db.Exec("DROP TABLE `redeem_logs`").Error
	s.NoError(err)
}

func (s *ModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `redemption_codes`").Error
	s.NoError(err)
	err = s.db.Exec("TRUNCATE TABLE `redeem_logs`").Error
	s.NoError(err)
}

func (s *ModuleTestSuite) newGinServer(handler *web.Handler) *egin.Component {
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: testID,
		}))
	})

	handler.PrivateRoutes(server.Engine)
	return server
}

func (s *ModuleTestSuite) TestConsumer_ConsumeOrderEvent() {
	t := s.T()

	testCases := []struct {
		name       string
		newMQFunc  func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ
		newSvcFunc func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service
		evt        event.OrderEvent
		after      func(t *testing.T, evt event.OrderEvent)

		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name: "消费完成订单消息成功_通过会员商品开通会员",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)

				mockProducer := mocks.NewMockProducer(ctrl)
				memberEvent := s.newMemberEventMessage(t, event.MemberEvent{
					Key:    evt.OrderSN,
					Uid:    evt.BuyerID,
					Days:   14,
					Biz:    "order",
					BizId:  1,
					Action: "购买会员商品",
				})
				mockProducer.EXPECT().Produce(gomock.Any(), memberEvent).Return(&mq.ProducerResult{}, nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service {
				t.Helper()

				mockOrderSvc := ordermocks.NewMockService(ctrl)

				mockOrderSvc.EXPECT().
					FindUserVisibleOrderByUIDAndSN(gomock.Any(), evt.BuyerID, evt.OrderSN).
					Return(order.Order{
						ID:               1,
						SN:               evt.OrderSN,
						BuyerID:          evt.BuyerID,
						OriginalTotalAmt: 330,
						RealTotalAmt:     330,
						Status:           order.StatusSuccess,
						Items: []order.Item{
							{
								SPU: order.SPU{
									ID:       1,
									Category: order.Category{Name: "member", Desc: "会员商品"},
								},
								SKU: order.SKU{
									ID:            1,
									SN:            "sku-sn-member-product",
									Attrs:         `{"days":7}`,
									OriginalPrice: 330,
									RealPrice:     330,
									Quantity:      2,
								},
							},
						},
					}, nil).Times(2)

				memberEventProducer, err := producer.NewMemberEventProducer(q)
				require.NoError(t, err)

				return service.NewService(mockOrderSvc, memberEventProducer, nil, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-member",
				BuyerID: 123456,
				SPUs: []event.SPU{
					{
						ID:       1,
						Category: "member",
					},
				},
			},
			errRequireFunc: require.NoError,
			after:          func(t *testing.T, evt event.OrderEvent) {},
		},
		// {
		// 	name: "消费完成订单消息成功_生成兑换码",
		// 	newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
		// 		t.Helper()
		//
		// 		mockMQ := mocks.NewMockMQ(ctrl)
		// 		mockConsumer := mocks.NewMockConsumer(ctrl)
		// 		mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)
		//
		// 		mockProducer := mocks.NewMockProducer(ctrl)
		//
		// 		mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
		// 		mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)
		// 		return mockMQ
		// 	},
		// 	newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service {
		// 		t.Helper()
		//
		// 		mockOrderSvc := ordermocks.NewMockService(ctrl)
		//
		// 		mockOrderSvc.EXPECT().
		// 			FindUserVisibleOrderByUIDAndSN(gomock.Any(), evt.BuyerID, evt.OrderSN).
		// 			Return(order.Order{
		// 				ID:               2,
		// 				SN:               evt.OrderSN,
		// 				BuyerID:          evt.BuyerID,
		// 				OriginalTotalAmt: 990,
		// 				RealTotalAmt:     990,
		// 				Status:           order.StatusSuccess,
		// 				Items: []order.Item{
		// 					{
		// 						SPU: order.SPU{
		// 							ID:       2,
		// 							Category: order.Category{Name: "code", Desc: "会员兑换码"},
		// 						},
		// 						SKU: order.SKU{
		// 							ID:            2,
		// 							SN:            "sku-sn-code-product",
		// 							Attrs:         `{"days":90}`,
		// 							OriginalPrice: 990,
		// 							RealPrice:     990,
		// 							Quantity:      1,
		// 						},
		// 					},
		// 				},
		// 			}, nil).Times(2)
		//
		// 		memberEventProducer, err := producer.NewMemberEventProducer(q)
		// 		require.NoError(t, err)
		//
		// 		return service.NewService(mockOrderSvc, memberEventProducer, sequencenumber.NewGenerator())
		// 	},
		// 	evt: event.OrderEvent{
		// 		OrderSN: "OrderSN-marketing-code",
		// 		BuyerID: 1234568,
		// 		SPUs: []event.SPU{
		// 			{
		// 				ID:       2,
		// 				Category: "code",
		// 			},
		// 		},
		// 	},
		// 	errRequireFunc: require.NoError,
		// 	after: func(t *testing.T, evt event.OrderEvent) {
		// 		t.Helper()
		//
		// 		// todo: 根据evt.BuyerID 找到 兑换码记录, 且状态为未使用
		// 	},
		// },
		{
			name: "消费完成订单消息成功_忽略不关心的完成订单消息",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)

				mockProducer := mocks.NewMockProducer(ctrl)
				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service {
				t.Helper()
				mockOrderSvc := ordermocks.NewMockService(ctrl)
				memberEventProducer, err := producer.NewMemberEventProducer(q)
				require.NoError(t, err)
				return service.NewService(mockOrderSvc, memberEventProducer, nil, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-other",
				BuyerID: 123457,
				SPUs: []event.SPU{
					{
						ID:       10,
						Category: "other",
					},
				},
			},
			errRequireFunc: require.NoError,
			after:          func(t *testing.T, evt event.OrderEvent) {},
		},
	}

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			q := tc.newMQFunc(t, ctrl, tc.evt)
			svc := tc.newSvcFunc(t, ctrl, tc.evt, q)
			c, err := consumer.NewOrderEventConsumer(svc, q)
			require.NoError(t, err)

			err = c.Consume(context.Background())
			tc.errRequireFunc(t, err)

			err = c.Consume(context.Background())
			tc.errRequireFunc(t, err)
		})
	}
}

func (s *ModuleTestSuite) newOrderEventMessage(t *testing.T, evt event.OrderEvent) *mq.Message {
	t.Helper()
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) newMemberEventMessage(t *testing.T, evt event.MemberEvent) *mq.Message {
	t.Helper()
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Key: []byte(evt.Key), Value: marshal}
}

func (s *ModuleTestSuite) TestHandler_ListRedemptionCode() {
	t := s.T()

	s.TearDownTest()

	total := 100
	for idx := 0; idx < total; idx++ {
		id := int64(100 + idx)
		status := domain.RedemptionCodeStatus(uint8(id)%2 + 1).ToUint8()
		codeEntity := dao.RedemptionCode{
			Id:      id,
			OwnerId: testID,
			OrderId: id,
			Code:    fmt.Sprintf("code-%d", id),
			Status:  status,
			Ctime:   time.Now().UnixMilli(),
			Utime:   time.Now().UnixMilli(),
		}
		_, err := s.dao.CreateRedemptionCode(context.Background(), codeEntity)
		require.NoError(t, err)
	}

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.Handler
		req            web.ListRedemptionCodesReq

		wantCode int
		wantResp test.Result[web.ListRedemptionCodesResp]
	}{
		{
			name: "获取成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.Handler {
				t.Helper()
				svc := service.NewService(nil, nil, sequencenumber.NewGenerator(), s.repo)
				return web.NewHandler(svc)
			},
			req: web.ListRedemptionCodesReq{
				Limit:  2,
				Offset: 0,
			},
			wantCode: 200,
			wantResp: test.Result[web.ListRedemptionCodesResp]{
				Data: web.ListRedemptionCodesResp{
					Total: int64(total),
					Codes: []web.RedemptionCode{
						{
							Code:   "code-199",
							Status: domain.RedemptionCodeStatusUsed.ToUint8(),
						},
						{
							Code:   "code-198",
							Status: domain.RedemptionCodeStatusUnused.ToUint8(),
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
				"/code/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.ListRedemptionCodesResp]()
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			s.assertListRedemptionCodesRespEqual(t, tc.wantResp.Data, recorder.MustScan().Data)
		})
	}
}

func (s *ModuleTestSuite) assertListRedemptionCodesRespEqual(t *testing.T, expected web.ListRedemptionCodesResp, actual web.ListRedemptionCodesResp) {
	assert.Equal(t, expected.Total, actual.Total)
	assert.Equal(t, len(expected.Codes), len(actual.Codes))
	for i := range expected.Codes {
		assert.NotZero(t, actual.Codes[i].Utime)
		actual.Codes[i].Utime = 0
	}
	assert.Equal(t, expected.Codes, actual.Codes)
}

// 生成流程
// 1. 生成成功 —— 查询到兑换码
// 2. 幂等重复消息返回第一次结果 —— orderEvent添加Key字段,表示唯一字段

// 兑换流程
// 1. 重复消息返回第一次结果
// 2. 兑换成功 —— 兑换码为已使用 + 发送“会员消息”
// 3. 兑换失败 —— 兑换码非法
// 4. 兑换失败 —— 超过限流次数1s一次

// 查询流程
// 1. 分页查询
