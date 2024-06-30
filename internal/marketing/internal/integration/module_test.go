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
	"io"
	"net/http"
	"sync"
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
	"github.com/ecodeclub/webook/internal/marketing/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/marketing/internal/service"
	"github.com/ecodeclub/webook/internal/marketing/internal/web"
	"github.com/ecodeclub/webook/internal/order"
	ordermocks "github.com/ecodeclub/webook/internal/order/mocks"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/ecodeclub/webook/internal/product"
	productmocks "github.com/ecodeclub/webook/internal/product/mocks"
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
	repo repository.MarketingRepository
}

func (s *ModuleTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	s.NoError(dao.InitTables(s.db))
	s.repo = repository.NewRepository(dao.NewGORMMarketingDAO(s.db), nil)
}

func (s *ModuleTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `redemption_codes`").Error
	s.NoError(err)
	err = s.db.Exec("DROP TABLE `redeem_logs`").Error
	s.NoError(err)
	err = s.db.Exec("DROP TABLE `generate_logs`").Error
	s.NoError(err)
	err = s.db.Exec("DROP TABLE `invitation_codes`").Error
	s.NoError(err)
	err = s.db.Exec("DROP TABLE `invitation_records`").Error
	s.NoError(err)
}

func (s *ModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `redemption_codes`").Error
	s.NoError(err)
	err = s.db.Exec("TRUNCATE TABLE `redeem_logs`").Error
	s.NoError(err)
	err = s.db.Exec("TRUNCATE TABLE `generate_logs`").Error
	s.NoError(err)
	err = s.db.Exec("TRUNCATE TABLE `invitation_codes`").Error
	s.NoError(err)
	err = s.db.Exec("TRUNCATE TABLE `invitation_records`").Error
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

func (s *ModuleTestSuite) newAdminGinServer(handler *web.AdminHandler) *egin.Component {
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
		name           string
		newMQFunc      func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ
		newSvcFunc     func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service
		evt            event.OrderEvent
		after          func(t *testing.T, evt event.OrderEvent)
		errRequireFunc require.ErrorAssertionFunc
	}{
		// 会员商品/兑换码
		{
			name: "消费完成订单消息成功_通过会员商品开通会员_单订单项_多个数量",
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
						OriginalTotalAmt: 660,
						RealTotalAmt:     660,
						Status:           order.StatusSuccess,
						Items: []order.Item{
							{
								SPU: order.SPU{
									ID:        1,
									Category0: "product",
									Category1: "member",
								},
								SKU: order.SKU{
									ID:            1,
									SN:            "sku-sn-member-product-1",
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

				eventKeyGenerator := func() string {
					return fmt.Sprintf("event-key-%s", evt.OrderSN)
				}
				return service.NewService(nil, mockOrderSvc, nil, nil, eventKeyGenerator, memberEventProducer, nil, nil, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-member",
				BuyerID: 123456,
				SPUs: []event.SPU{
					{
						ID:        1,
						Category0: "product",
						Category1: "member",
					},
				},
			},
			errRequireFunc: require.NoError,
			after:          func(t *testing.T, evt event.OrderEvent) {},
		},
		{
			name: "消费完成订单消息成功_通过会员商品开通会员_多订单项_混合数量",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)

				mockProducer := mocks.NewMockProducer(ctrl)
				memberEvent := s.newMemberEventMessage(t, event.MemberEvent{
					Key:    evt.OrderSN,
					Uid:    evt.BuyerID,
					Days:   21,
					Biz:    "order",
					BizId:  2,
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
						ID:               2,
						SN:               evt.OrderSN,
						BuyerID:          evt.BuyerID,
						OriginalTotalAmt: 990,
						RealTotalAmt:     990,
						Status:           order.StatusSuccess,
						Items: []order.Item{
							{
								SPU: order.SPU{
									ID:        1,
									Category0: "product",
									Category1: "member",
								},
								SKU: order.SKU{
									ID:            2,
									SN:            "sku-sn-member-product-2",
									Attrs:         `{"days":7}`,
									OriginalPrice: 330,
									RealPrice:     330,
									Quantity:      2,
								},
							},
							{
								SPU: order.SPU{
									ID:        1,
									Category0: "product",
									Category1: "member",
								},
								SKU: order.SKU{
									ID:            3,
									SN:            "sku-sn-member-product-3",
									Attrs:         `{"days":7}`,
									OriginalPrice: 330,
									RealPrice:     330,
									Quantity:      1,
								},
							},
						},
					}, nil).Times(2)

				memberEventProducer, err := producer.NewMemberEventProducer(q)
				require.NoError(t, err)

				eventKeyGenerator := func() string {
					return fmt.Sprintf("event-key-%s", evt.OrderSN)
				}
				return service.NewService(nil, mockOrderSvc, nil, nil, eventKeyGenerator, memberEventProducer, nil, nil, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-member-2",
				BuyerID: 123456,
				SPUs: []event.SPU{
					{
						ID:        1,
						Category0: "product",
						Category1: "member",
					},
					{
						ID:        1,
						Category0: "product",
						Category1: "member",
					},
				},
			},
			errRequireFunc: require.NoError,
			after:          func(t *testing.T, evt event.OrderEvent) {},
		},
		{
			name: "消费完成订单消息成功_生成会员商品兑换码_单订单项_多个数量",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service {
				t.Helper()

				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().
					FindUserVisibleOrderByUIDAndSN(gomock.Any(), evt.BuyerID, evt.OrderSN).
					Return(order.Order{
						ID:               3,
						SN:               evt.OrderSN,
						BuyerID:          evt.BuyerID,
						OriginalTotalAmt: 1980,
						RealTotalAmt:     1980,
						Status:           order.StatusSuccess,
						Items: []order.Item{
							{
								SPU: order.SPU{
									ID:        2,
									Category0: "code",
									Category1: "member",
								},
								SKU: order.SKU{
									ID:            4,
									SN:            "sku-sn-4",
									Name:          "sku-name-4",
									Attrs:         `{"days":90}`,
									OriginalPrice: 990,
									RealPrice:     990,
									Quantity:      2,
								},
							},
						},
					}, nil).Times(2)

				return service.NewService(s.repo, mockOrderSvc, nil, s.getRedemptionCodeGenerator(sequencenumber.NewGenerator()),
					nil, nil, nil, nil, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-code-member-1",
				BuyerID: 1234568,
				SPUs: []event.SPU{
					{
						ID:        2,
						Category0: "code",
						Category1: "member",
					},
				},
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, evt event.OrderEvent) {
				t.Helper()
				codes, err := s.repo.FindRedemptionCodesByUID(context.Background(), evt.BuyerID, 0, 10)
				require.NoError(t, err)
				oid := int64(3)
				skuId := int64(4)
				code := s.newMemberRedemptionCodeDomain(evt.BuyerID, oid, skuId)
				code.Attrs.SKU.Attrs = `{"days":90}`
				code.Code = ""
				s.assertRedemptionCodeEqual(t, []domain.RedemptionCode{code, code}, codes)
			},
		},
		{
			name: "消费完成订单消息成功_生成会员商品兑换码_多订单项_混合数量",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service {
				t.Helper()

				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().
					FindUserVisibleOrderByUIDAndSN(gomock.Any(), evt.BuyerID, evt.OrderSN).
					Return(order.Order{
						ID:               4,
						SN:               evt.OrderSN,
						BuyerID:          evt.BuyerID,
						OriginalTotalAmt: 2310,
						RealTotalAmt:     2310,
						Status:           order.StatusSuccess,
						Items: []order.Item{
							{
								SPU: order.SPU{
									ID:        2,
									Category0: "code",
									Category1: "member",
								},
								SKU: order.SKU{
									ID:            4,
									SN:            "sku-sn-4",
									Name:          "sku-name-4",
									Attrs:         `{"days":90}`,
									OriginalPrice: 990,
									RealPrice:     990,
									Quantity:      2,
								},
							},
							{
								SPU: order.SPU{
									ID:        2,
									Category0: "code",
									Category1: "member",
								},
								SKU: order.SKU{
									ID:            5,
									SN:            "sku-sn-5",
									Name:          "sku-name-5",
									Attrs:         `{"days":30}`,
									OriginalPrice: 330,
									RealPrice:     330,
									Quantity:      1,
								},
							},
						},
					}, nil).Times(2)

				return service.NewService(s.repo, mockOrderSvc, nil, s.getRedemptionCodeGenerator(sequencenumber.NewGenerator()),
					nil, nil, nil, nil, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-code-member-2",
				BuyerID: 1234569,
				SPUs: []event.SPU{
					{
						ID:        2,
						Category0: "code",
						Category1: "member",
					},
					{
						ID:        2,
						Category0: "code",
						Category1: "member",
					},
				},
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, evt event.OrderEvent) {
				t.Helper()
				codes, err := s.repo.FindRedemptionCodesByUID(context.Background(), evt.BuyerID, 0, 10)
				require.NoError(t, err)
				oid := int64(4)
				skuId := int64(4)
				code90 := s.newMemberRedemptionCodeDomain(evt.BuyerID, oid, skuId)
				code90.Attrs.SKU.Attrs = `{"days":90}`
				code90.Code = ""
				skuId = int64(5)
				code30 := s.newMemberRedemptionCodeDomain(evt.BuyerID, oid, skuId)
				code30.Code = ""
				s.assertRedemptionCodeEqual(t, []domain.RedemptionCode{code90, code90, code30}, codes)
			},
		},

		// 面试项目商品/兑换码
		{
			name: "消费完成订单消息成功_通过项目商品开通项目权限_单订单项",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)

				mockProducer := mocks.NewMockProducer(ctrl)
				permissionEvent := s.newPermissionEventMessage(t, event.PermissionEvent{
					Uid:    evt.BuyerID,
					Biz:    "project",
					BizIds: []int64{123},
					Action: "购买项目商品",
				})
				mockProducer.EXPECT().Produce(gomock.Any(), permissionEvent).Return(&mq.ProducerResult{}, nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				mockMQ.EXPECT().Producer(event.PermissionEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service {
				t.Helper()

				mockOrderSvc := ordermocks.NewMockService(ctrl)

				mockOrderSvc.EXPECT().
					FindUserVisibleOrderByUIDAndSN(gomock.Any(), evt.BuyerID, evt.OrderSN).
					Return(order.Order{
						ID:               101,
						SN:               evt.OrderSN,
						BuyerID:          evt.BuyerID,
						OriginalTotalAmt: 19800,
						RealTotalAmt:     19800,
						Status:           order.StatusSuccess,
						Items: []order.Item{
							{
								SPU: order.SPU{
									ID:        3,
									Category0: "product",
									Category1: "project",
								},
								SKU: order.SKU{
									ID:            10,
									SN:            "sku-sn-project-product-1",
									Attrs:         `{"projectId": 123}`,
									OriginalPrice: 9900,
									RealPrice:     9900,
									Quantity:      2,
								},
							},
						},
					}, nil).Times(2)

				permissionEventProducer, err := producer.NewPermissionEventProducer(q)
				require.NoError(t, err)

				eventKeyGenerator := func() string {
					return fmt.Sprintf("event-key-%s", evt.OrderSN)
				}
				return service.NewService(nil, mockOrderSvc, nil, nil, eventKeyGenerator, nil, nil, permissionEventProducer, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-project-101",
				BuyerID: 45612378,
				SPUs: []event.SPU{
					{
						ID:        3,
						Category0: "product",
						Category1: "project",
					},
				},
			},
			errRequireFunc: require.NoError,
			after:          func(t *testing.T, evt event.OrderEvent) {},
		},
		{
			name: "消费完成订单消息成功_通过项目商品开通项目权限_多订单项",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)

				mockProducer := mocks.NewMockProducer(ctrl)
				permissionEvent := s.newPermissionEventMessage(t, event.PermissionEvent{
					Uid:    evt.BuyerID,
					Biz:    "project",
					BizIds: []int64{234, 345},
					Action: "购买项目商品",
				})
				mockProducer.EXPECT().Produce(gomock.Any(), permissionEvent).Return(&mq.ProducerResult{}, nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				mockMQ.EXPECT().Producer(event.PermissionEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service {
				t.Helper()

				mockOrderSvc := ordermocks.NewMockService(ctrl)

				mockOrderSvc.EXPECT().
					FindUserVisibleOrderByUIDAndSN(gomock.Any(), evt.BuyerID, evt.OrderSN).
					Return(order.Order{
						ID:               102,
						SN:               evt.OrderSN,
						BuyerID:          evt.BuyerID,
						OriginalTotalAmt: 9900 * 3,
						RealTotalAmt:     9900 * 3,
						Status:           order.StatusSuccess,
						Items: []order.Item{
							{
								SPU: order.SPU{
									ID:        3,
									Category0: "product",
									Category1: "project",
								},
								SKU: order.SKU{
									ID:            12,
									SN:            "sku-sn-project-product-2",
									Attrs:         `{"projectId": 234}`,
									OriginalPrice: 9900,
									RealPrice:     9900,
									Quantity:      2,
								},
							},
							{
								SPU: order.SPU{
									ID:        3,
									Category0: "product",
									Category1: "project",
								},
								SKU: order.SKU{
									ID:            13,
									SN:            "sku-sn-project-product-3",
									Attrs:         `{"projectId": 345}`,
									OriginalPrice: 9900,
									RealPrice:     9900,
									Quantity:      1,
								},
							},
						},
					}, nil).Times(2)

				permissionEventProducer, err := producer.NewPermissionEventProducer(q)
				require.NoError(t, err)

				eventKeyGenerator := func() string {
					return fmt.Sprintf("event-key-%s", evt.OrderSN)
				}
				return service.NewService(nil, mockOrderSvc, nil, nil, eventKeyGenerator, nil, nil, permissionEventProducer, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-project-102",
				BuyerID: 456123789,
				SPUs: []event.SPU{
					{
						ID:        3,
						Category0: "product",
						Category1: "project",
					},
				},
			},
			errRequireFunc: require.NoError,
			after:          func(t *testing.T, evt event.OrderEvent) {},
		},
		{
			name: "消费完成订单消息成功_生成项目商品兑换码_单订单项_多个数量",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service {
				t.Helper()

				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().
					FindUserVisibleOrderByUIDAndSN(gomock.Any(), evt.BuyerID, evt.OrderSN).
					Return(order.Order{
						ID:               103,
						SN:               evt.OrderSN,
						BuyerID:          evt.BuyerID,
						OriginalTotalAmt: 1980,
						RealTotalAmt:     1980,
						Status:           order.StatusSuccess,
						Items: []order.Item{
							{
								SPU: order.SPU{
									ID:        4,
									Category0: "code",
									Category1: "project",
								},
								SKU: order.SKU{
									ID:            14,
									SN:            "sku-sn-14",
									Name:          "sku-name-14",
									Attrs:         `{"projectId": 456}`,
									OriginalPrice: 990,
									RealPrice:     990,
									Quantity:      2,
								},
							},
						},
					}, nil).Times(2)

				return service.NewService(s.repo, mockOrderSvc, nil, s.getRedemptionCodeGenerator(sequencenumber.NewGenerator()),
					nil, nil, nil, nil, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-code-project-103",
				BuyerID: 78965421,
				SPUs: []event.SPU{
					{
						ID:        4,
						Category0: "code",
						Category1: "project",
					},
				},
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, evt event.OrderEvent) {
				t.Helper()
				codes, err := s.repo.FindRedemptionCodesByUID(context.Background(), evt.BuyerID, 0, 10)
				require.NoError(t, err)
				oid := int64(103)
				skuId := int64(14)
				code := s.newProjectRedemptionCodeDomain(evt.BuyerID, oid, skuId)
				code.Code = ""
				code.Attrs.SKU.Attrs = `{"projectId": 456}`
				s.assertRedemptionCodeEqual(t, []domain.RedemptionCode{code, code}, codes)
			},
		},
		{
			name: "消费完成订单消息成功_生成项目商品兑换码_多订单项_混合数量",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service {
				t.Helper()

				mockOrderSvc := ordermocks.NewMockService(ctrl)
				mockOrderSvc.EXPECT().
					FindUserVisibleOrderByUIDAndSN(gomock.Any(), evt.BuyerID, evt.OrderSN).
					Return(order.Order{
						ID:               104,
						SN:               evt.OrderSN,
						BuyerID:          evt.BuyerID,
						OriginalTotalAmt: 990 * 3,
						RealTotalAmt:     990 * 3,
						Status:           order.StatusSuccess,
						Items: []order.Item{
							{
								SPU: order.SPU{
									ID:        4,
									Category0: "code",
									Category1: "project",
								},
								SKU: order.SKU{
									ID:            14,
									SN:            "sku-sn-14",
									Name:          "sku-name-14",
									Attrs:         `{"projectId": 456}`,
									OriginalPrice: 990,
									RealPrice:     990,
									Quantity:      2,
								},
							},
							{
								SPU: order.SPU{
									ID:        4,
									Category0: "code",
									Category1: "project",
								},
								SKU: order.SKU{
									ID:            15,
									SN:            "sku-sn-15",
									Name:          "sku-name-15",
									Attrs:         `{"projectId": 789}`,
									OriginalPrice: 990,
									RealPrice:     990,
									Quantity:      1,
								},
							},
						},
					}, nil).Times(2)

				return service.NewService(s.repo, mockOrderSvc, nil, s.getRedemptionCodeGenerator(sequencenumber.NewGenerator()),
					nil, nil, nil, nil, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-code-project-104",
				BuyerID: 78965431,
				SPUs: []event.SPU{
					{
						ID:        4,
						Category0: "code",
						Category1: "project",
					},
				},
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, evt event.OrderEvent) {
				t.Helper()
				codes, err := s.repo.FindRedemptionCodesByUID(context.Background(), evt.BuyerID, 0, 10)
				require.NoError(t, err)
				oid := int64(104)
				skuId := int64(14)
				code14 := s.newProjectRedemptionCodeDomain(evt.BuyerID, oid, skuId)
				code14.Code = ""
				code14.Attrs.SKU.Attrs = `{"projectId": 456}`
				skuId = int64(15)
				code15 := s.newProjectRedemptionCodeDomain(evt.BuyerID, oid, skuId)
				code15.Code = ""
				code15.Attrs.SKU.Attrs = `{"projectId": 789}`
				s.assertRedemptionCodeEqual(t, []domain.RedemptionCode{code14, code14, code15}, codes)
			},
		},

		// 面试服务商品
		{
			name: "消费完成订单消息成功_通过面试服务商品发送企业微信信息_单订单项",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newOrderEventMessage(t, evt), nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service {
				t.Helper()

				mockOrderSvc := ordermocks.NewMockService(ctrl)

				orderId := 201
				mockOrderSvc.EXPECT().
					FindUserVisibleOrderByUIDAndSN(gomock.Any(), evt.BuyerID, evt.OrderSN).
					Return(order.Order{
						ID:               int64(orderId),
						SN:               evt.OrderSN,
						BuyerID:          evt.BuyerID,
						OriginalTotalAmt: 50000,
						RealTotalAmt:     50000,
						Status:           order.StatusSuccess,
						Items: []order.Item{
							{
								SPU: order.SPU{
									ID:        5,
									Category0: "product",
									Category1: "service",
								},
								SKU: order.SKU{
									ID:            16,
									SN:            "sku-sn-service-product-1",
									Attrs:         ``,
									OriginalPrice: 500 * 100,
									RealPrice:     500 * 100,
									Quantity:      1,
								},
							},
						},
					}, nil).Times(2)

				postFunc := func(url, contentType string, body io.Reader) (resp *http.Response, err error) {

					require.NotZero(t, url)

					require.Equal(t, "application/json", contentType)

					bs, err := io.ReadAll(body)
					require.NoError(t, err)
					type Message struct {
						MsgType string              `json:"msgtype"`
						Text    event.QYWechatEvent `json:"text"`
					}
					var msg Message
					err = json.Unmarshal(bs, &msg)
					require.NoError(t, err)
					require.Equal(t, "text", msg.MsgType)
					require.Contains(t, msg.Text.Content, fmt.Sprintf("%d", orderId))

					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(body)}, nil
				}

				qyWechatEventProducer := producer.NewQYWeChatEventProducer("whatever", postFunc)
				return service.NewService(nil, mockOrderSvc, nil, nil, nil, nil, nil, nil, qyWechatEventProducer)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-service-101",
				BuyerID: 887655432,
				SPUs: []event.SPU{
					{
						ID:        5,
						Category0: "product",
						Category1: "service",
					},
				},
			},
			errRequireFunc: require.NoError,
			after:          func(t *testing.T, evt event.OrderEvent) {},
		},

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
				return service.NewService(nil, mockOrderSvc, nil, nil, nil, memberEventProducer, nil, nil, nil)
			},
			evt: event.OrderEvent{
				OrderSN: "OrderSN-marketing-other",
				BuyerID: 123457,
				SPUs: []event.SPU{
					{
						ID:        10,
						Category0: "other",
						Category1: "other",
					},
				},
			},
			errRequireFunc: require.NoError,
			after:          func(t *testing.T, evt event.OrderEvent) {},
		},
	}

	for _, tc := range testCases {
		tc := tc
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

			tc.after(t, tc.evt)
		})
	}
}

func (s *ModuleTestSuite) assertRedemptionCodeEqual(t *testing.T, expected []domain.RedemptionCode, actual []domain.RedemptionCode) {
	for i, c := range actual {
		assert.NotZero(t, c.ID)
		assert.NotZero(t, c.Code)
		assert.NotZero(t, c.Ctime)
		assert.NotZero(t, c.Utime)
		actual[i].ID = 0
		actual[i].Code = ""
		actual[i].Ctime = 0
		actual[i].Utime = 0
	}
	assert.Equal(t, expected, actual)
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
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) newPermissionEventMessage(t *testing.T, evt event.PermissionEvent) *mq.Message {
	t.Helper()
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) TestConsumer_ConsumeUserRegistrationEvent() {
	t := s.T()

	testCases := []struct {
		name           string
		newMQFunc      func(t *testing.T, ctrl *gomock.Controller, evt event.UserRegistrationEvent) mq.MQ
		newSvcFunc     func(t *testing.T, ctrl *gomock.Controller, evt event.UserRegistrationEvent, q mq.MQ) service.Service
		evt            event.UserRegistrationEvent
		errRequireFunc require.ErrorAssertionFunc
		after          func(t *testing.T, evt event.UserRegistrationEvent)
	}{
		{
			name: "消费注册消息成功_为注册者开通会员",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.UserRegistrationEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newUserRegistrationEventMessage(t, evt), nil).Times(2)

				mockProducer := mocks.NewMockProducer(ctrl)
				endAtDate := time.Date(2024, 9, 30, 23, 59, 59, 0, time.UTC)
				memberEvent := s.newMemberEventMessage(t, event.MemberEvent{
					Key:    fmt.Sprintf("user-registration-%d", evt.Uid),
					Uid:    evt.Uid,
					Days:   uint64(time.Until(endAtDate) / (24 * time.Hour)),
					Biz:    "user",
					BizId:  evt.Uid,
					Action: "注册福利",
				})
				mockProducer.EXPECT().Produce(gomock.Any(), memberEvent).Return(&mq.ProducerResult{}, nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.UserRegistrationEvent, q mq.MQ) service.Service {
				t.Helper()

				memberEventProducer, err := producer.NewMemberEventProducer(q)
				require.NoError(t, err)

				repo := repository.NewRepository(dao.NewGORMMarketingDAO(s.db), cache.NewInvitationCodeECache(testioc.InitCache(), time.Minute*10))

				return service.NewService(repo, nil, nil, nil, nil, memberEventProducer, nil, nil, nil)
			},
			evt: event.UserRegistrationEvent{
				Uid: testID,
			},
			errRequireFunc: require.NoError,
			after:          func(t *testing.T, evt event.UserRegistrationEvent) {},
		},
		{
			name: "消费注册消息成功_为注册者开通会员_为邀请者增加积分",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.UserRegistrationEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newUserRegistrationEventMessage(t, evt), nil).Times(2)

				mockProducer := mocks.NewMockProducer(ctrl)
				endAtDate := time.Date(2024, 9, 30, 23, 59, 59, 0, time.UTC)
				memberEvent := s.newMemberEventMessage(t, event.MemberEvent{
					Key:    fmt.Sprintf("user-registration-%d", evt.Uid),
					Uid:    evt.Uid,
					Days:   uint64(time.Until(endAtDate) / (24 * time.Hour)),
					Biz:    "user",
					BizId:  evt.Uid,
					Action: "注册福利",
				})
				mockProducer.EXPECT().Produce(gomock.Any(), memberEvent).Return(&mq.ProducerResult{}, nil).Times(2)

				inviterId := int64(345691)
				creditsAwarded := uint64(500)
				creditEvent := s.newCreditEventMessage(t, event.CreditIncreaseEvent{
					Key:    fmt.Sprintf("inviteeId-%d", evt.Uid),
					Uid:    inviterId,
					Amount: creditsAwarded,
					Biz:    "user",
					BizId:  evt.Uid,
					Action: "邀请奖励",
				})
				mockProducer.EXPECT().Produce(gomock.Any(), creditEvent).Return(&mq.ProducerResult{}, nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)
				mockMQ.EXPECT().Producer(event.CreditEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.UserRegistrationEvent, q mq.MQ) service.Service {
				t.Helper()

				memberEventProducer, err := producer.NewMemberEventProducer(q)
				require.NoError(t, err)

				creditEventProducer, err := producer.NewCreditEventProducer(q)
				require.NoError(t, err)

				repo := repository.NewRepository(dao.NewGORMMarketingDAO(s.db), cache.NewInvitationCodeECache(testioc.InitCache(), time.Minute*10))

				expectedCode := domain.InvitationCode{
					Uid:  345691,
					Code: evt.InvitationCode,
				}
				_, err = repo.CreateInvitationCode(context.Background(), expectedCode)
				require.NoError(t, err)

				return service.NewService(repo, nil, nil, nil, nil, memberEventProducer, creditEventProducer, nil, nil)
			},
			evt: event.UserRegistrationEvent{
				Uid:            testID,
				InvitationCode: "registration-invitation-code-345691",
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, evt event.UserRegistrationEvent) {
				t.Helper()
				repo := repository.NewRepository(dao.NewGORMMarketingDAO(s.db), cache.NewInvitationCodeECache(testioc.InitCache(), time.Minute*10))
				inviterId := int64(345691)

				c := testioc.InitCache()
				_, err := c.Delete(context.Background(), fmt.Sprintf("marketing:invitation-code:user:%d", inviterId))
				require.NoError(t, err)

				record, err := repo.FindInvitationRecord(context.Background(), inviterId, testID, evt.InvitationCode)
				require.NoError(t, err)
				creditsAwarded := uint64(500)
				require.Equal(t, domain.InvitationRecord{
					InviterId: inviterId,
					InviteeId: testID,
					Code:      evt.InvitationCode,
					Attrs:     domain.InvitationRecordAttrs{Credits: creditsAwarded},
				}, record)
			},
		},
		{
			name: "消费注册消息成功_为注册者开通会员_邀请码找不则忽略",
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.UserRegistrationEvent) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)
				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newUserRegistrationEventMessage(t, evt), nil).Times(2)

				mockProducer := mocks.NewMockProducer(ctrl)
				endAtDate := time.Date(2024, 9, 30, 23, 59, 59, 0, time.UTC)
				memberEvent := s.newMemberEventMessage(t, event.MemberEvent{
					Key:    fmt.Sprintf("user-registration-%d", evt.Uid),
					Uid:    evt.Uid,
					Days:   uint64(time.Until(endAtDate) / (24 * time.Hour)),
					Biz:    "user",
					BizId:  evt.Uid,
					Action: "注册福利",
				})
				mockProducer.EXPECT().Produce(gomock.Any(), memberEvent).Return(&mq.ProducerResult{}, nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)
				mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)
				mockMQ.EXPECT().Producer(event.CreditEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newSvcFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.UserRegistrationEvent, q mq.MQ) service.Service {
				t.Helper()

				memberEventProducer, err := producer.NewMemberEventProducer(q)
				require.NoError(t, err)

				creditEventProducer, err := producer.NewCreditEventProducer(q)
				require.NoError(t, err)

				repo := repository.NewRepository(dao.NewGORMMarketingDAO(s.db), cache.NewInvitationCodeECache(testioc.InitCache(), time.Minute*10))

				return service.NewService(repo, nil, nil, nil, nil, memberEventProducer, creditEventProducer, nil, nil)
			},
			evt: event.UserRegistrationEvent{
				Uid:            testID,
				InvitationCode: "invalid-registration-invitation-code",
			},
			errRequireFunc: require.NoError,
			after:          func(t *testing.T, evt event.UserRegistrationEvent) {},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			q := tc.newMQFunc(t, ctrl, tc.evt)
			svc := tc.newSvcFunc(t, ctrl, tc.evt, q)
			c, err := consumer.NewUserRegistrationEventConsumer(svc, q)
			require.NoError(t, err)

			err = c.Consume(context.Background())
			tc.errRequireFunc(t, err)

			err = c.Consume(context.Background())
			tc.errRequireFunc(t, err)

			if err != nil {
				return
			}

			tc.after(t, tc.evt)
		})
	}
}

func (s *ModuleTestSuite) newUserRegistrationEventMessage(t *testing.T, evt event.UserRegistrationEvent) *mq.Message {
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) newCreditEventMessage(t *testing.T, evt event.CreditIncreaseEvent) *mq.Message {
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) TestHandler_RedeemRedemptionCode() {
	t := s.T()

	testCases := []struct {
		name string

		req            web.RedeemRedemptionCodeReq
		before         func(t *testing.T, req web.RedeemRedemptionCodeReq) domain.RedemptionCode
		newEvtFunc     func(code domain.RedemptionCode) any
		newMQFunc      func(t *testing.T, ctrl *gomock.Controller, eve any) mq.MQ
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller, q mq.MQ) *web.Handler
		after          func(t *testing.T, code domain.RedemptionCode)
		wantCode       int
		wantResp       test.Result[any]
	}{
		{
			name: "兑换会员成功_所有者兑换",
			req: web.RedeemRedemptionCodeReq{
				Code: "redemption-code-member-001",
			},
			before: func(t *testing.T, req web.RedeemRedemptionCodeReq) domain.RedemptionCode {
				t.Helper()
				oid := int64(1101)
				skuId := int64(4)
				code := s.newMemberRedemptionCodeDomain(testID, oid, skuId)
				code.Code = req.Code
				ids, err := s.repo.CreateRedemptionCodes(context.Background(), []domain.RedemptionCode{
					code,
				})
				require.NoError(t, err)
				code.ID = ids[0]
				return code
			},
			newEvtFunc: func(code domain.RedemptionCode) any {
				return event.MemberEvent{
					Key:    fmt.Sprintf("code-member-%d", code.ID),
					Uid:    code.OwnerID,
					Days:   30,
					Biz:    code.Biz,
					BizId:  code.BizId,
					Action: "兑换会员商品",
				}
			},
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, eve any) mq.MQ {
				t.Helper()

				evt, ok := eve.(event.MemberEvent)
				require.True(t, ok)

				mockMQ := mocks.NewMockMQ(ctrl)

				mockProducer := mocks.NewMockProducer(ctrl)
				memberEvent := s.newMemberEventMessage(t, evt)
				mockProducer.EXPECT().Produce(gomock.Any(), gomock.Eq(memberEvent)).Return(&mq.ProducerResult{}, nil)

				mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller, q mq.MQ) *web.Handler {
				t.Helper()

				mockProductSvc := productmocks.NewMockService(ctrl)

				mockOrderSvc := ordermocks.NewMockService(ctrl)

				memberEventProducer, err := producer.NewMemberEventProducer(q)
				require.NoError(t, err)

				svc := service.NewService(s.repo, mockOrderSvc, mockProductSvc, nil, nil, memberEventProducer, nil, nil, nil)
				return web.NewHandler(svc)
			},

			after: func(t *testing.T, code domain.RedemptionCode) {
				t.Helper()
				c, err := s.repo.FindRedemptionCode(context.Background(), code.Code)
				require.NoError(t, err)
				require.Equal(t, code.Code, c.Code)
				require.Equal(t, domain.RedemptionCodeStatusUsed, c.Status)
				require.NotEqual(t, c.Utime, c.Ctime)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
		},
		{
			name: "兑换会员成功_非所有者兑换",
			req: web.RedeemRedemptionCodeReq{
				Code: "redemption-code-member-002",
			},
			before: func(t *testing.T, req web.RedeemRedemptionCodeReq) domain.RedemptionCode {
				t.Helper()
				oid := int64(1102)
				skuId := int64(4)
				code := s.newMemberRedemptionCodeDomain(8922391, oid, skuId)
				code.Attrs.SKU.Attrs = `{"days":60}`
				code.Code = req.Code
				ids, err := s.repo.CreateRedemptionCodes(context.Background(), []domain.RedemptionCode{
					code,
				})
				require.NoError(t, err)
				code.ID = ids[0]
				return code
			},
			newEvtFunc: func(code domain.RedemptionCode) any {
				oid := int64(1102)
				return event.MemberEvent{
					Key:    fmt.Sprintf("code-member-%d", code.ID),
					Uid:    testID,
					Days:   60,
					Biz:    "order",
					BizId:  oid,
					Action: "兑换会员商品",
				}
			},
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, eve any) mq.MQ {
				t.Helper()

				evt, ok := eve.(event.MemberEvent)
				require.True(t, ok)

				mockMQ := mocks.NewMockMQ(ctrl)

				mockProducer := mocks.NewMockProducer(ctrl)
				memberEvent := s.newMemberEventMessage(t, evt)
				mockProducer.EXPECT().Produce(gomock.Any(), gomock.Eq(memberEvent)).Return(&mq.ProducerResult{}, nil)

				mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller, q mq.MQ) *web.Handler {
				t.Helper()

				mockProductSvc := productmocks.NewMockService(ctrl)

				mockOrderSvc := ordermocks.NewMockService(ctrl)

				memberEventProducer, err := producer.NewMemberEventProducer(q)
				require.NoError(t, err)

				svc := service.NewService(s.repo, mockOrderSvc, mockProductSvc, nil, nil, memberEventProducer, nil, nil, nil)
				return web.NewHandler(svc)
			},

			after: func(t *testing.T, code domain.RedemptionCode) {
				t.Helper()
				c, err := s.repo.FindRedemptionCode(context.Background(), code.Code)
				require.NoError(t, err)
				require.Equal(t, code.Code, c.Code)
				require.Equal(t, domain.RedemptionCodeStatusUsed, c.Status)
				require.NotEqual(t, c.Utime, c.Ctime)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
		},

		{
			name: "兑换项目成功_所有者兑换",
			req: web.RedeemRedemptionCodeReq{
				Code: "redemption-code-project-001",
			},
			before: func(t *testing.T, req web.RedeemRedemptionCodeReq) domain.RedemptionCode {
				t.Helper()
				oid := int64(2101)
				skuId := int64(14)
				code := s.newProjectRedemptionCodeDomain(testID, oid, skuId)
				code.Code = req.Code
				code.Attrs.SKU.Attrs = `{"projectId": 456}`
				ids, err := s.repo.CreateRedemptionCodes(context.Background(), []domain.RedemptionCode{
					code,
				})
				require.NoError(t, err)
				code.ID = ids[0]
				return code
			},
			newEvtFunc: func(code domain.RedemptionCode) any {
				return event.PermissionEvent{
					Uid:    code.OwnerID,
					Biz:    code.Type,
					BizIds: []int64{456},
					Action: "兑换项目商品",
				}
			},
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, eve any) mq.MQ {
				t.Helper()

				evt, ok := eve.(event.PermissionEvent)
				require.True(t, ok)

				mockMQ := mocks.NewMockMQ(ctrl)

				mockProducer := mocks.NewMockProducer(ctrl)
				memberEvent := s.newPermissionEventMessage(t, evt)
				mockProducer.EXPECT().Produce(gomock.Any(), gomock.Eq(memberEvent)).Return(&mq.ProducerResult{}, nil)

				mockMQ.EXPECT().Producer(event.PermissionEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller, q mq.MQ) *web.Handler {
				t.Helper()

				mockProductSvc := productmocks.NewMockService(ctrl)

				mockOrderSvc := ordermocks.NewMockService(ctrl)

				permissionEventProducer, err := producer.NewPermissionEventProducer(q)
				require.NoError(t, err)

				svc := service.NewService(s.repo, mockOrderSvc, mockProductSvc, nil, nil, nil, nil, permissionEventProducer, nil)
				return web.NewHandler(svc)
			},

			after: func(t *testing.T, code domain.RedemptionCode) {
				t.Helper()
				c, err := s.repo.FindRedemptionCode(context.Background(), code.Code)
				require.NoError(t, err)
				require.Equal(t, code.Code, c.Code)
				require.Equal(t, domain.RedemptionCodeStatusUsed, c.Status)
				require.NotEqual(t, c.Utime, c.Ctime)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
		},
		{
			name: "兑换项目成功_非所有者兑换",
			req: web.RedeemRedemptionCodeReq{
				Code: "redemption-code-project-002",
			},
			before: func(t *testing.T, req web.RedeemRedemptionCodeReq) domain.RedemptionCode {
				t.Helper()
				oid := int64(2102)
				skuId := int64(15)
				code := s.newProjectRedemptionCodeDomain(45672928, oid, skuId)
				code.Code = req.Code
				code.Attrs.SKU.Attrs = `{"projectId": 789}`
				ids, err := s.repo.CreateRedemptionCodes(context.Background(), []domain.RedemptionCode{
					code,
				})
				require.NoError(t, err)
				code.ID = ids[0]
				return code
			},
			newEvtFunc: func(code domain.RedemptionCode) any {
				return event.PermissionEvent{
					Uid:    testID,
					Biz:    code.Type,
					BizIds: []int64{789},
					Action: "兑换项目商品",
				}
			},
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, eve any) mq.MQ {
				t.Helper()

				evt, ok := eve.(event.PermissionEvent)
				require.True(t, ok)

				mockMQ := mocks.NewMockMQ(ctrl)

				mockProducer := mocks.NewMockProducer(ctrl)
				memberEvent := s.newPermissionEventMessage(t, evt)
				mockProducer.EXPECT().Produce(gomock.Any(), gomock.Eq(memberEvent)).Return(&mq.ProducerResult{}, nil)

				mockMQ.EXPECT().Producer(event.PermissionEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller, q mq.MQ) *web.Handler {
				t.Helper()

				mockProductSvc := productmocks.NewMockService(ctrl)

				mockOrderSvc := ordermocks.NewMockService(ctrl)

				permissionEventProducer, err := producer.NewPermissionEventProducer(q)
				require.NoError(t, err)

				svc := service.NewService(s.repo, mockOrderSvc, mockProductSvc, nil, nil, nil, nil, permissionEventProducer, nil)
				return web.NewHandler(svc)
			},

			after: func(t *testing.T, code domain.RedemptionCode) {
				t.Helper()
				c, err := s.repo.FindRedemptionCode(context.Background(), code.Code)
				require.NoError(t, err)
				require.Equal(t, code.Code, c.Code)
				require.Equal(t, domain.RedemptionCodeStatusUsed, c.Status)
				require.NotEqual(t, c.Utime, c.Ctime)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
		},

		{
			name: "兑换失败_兑换码已使用",
			req: web.RedeemRedemptionCodeReq{
				Code: "redemption-code-all-003",
			},
			before: func(t *testing.T, req web.RedeemRedemptionCodeReq) domain.RedemptionCode {
				t.Helper()
				oid := int64(1103)
				skuId := int64(12)
				code := s.newMemberRedemptionCodeDomain(7622391, oid, skuId)
				code.Attrs.SKU.Attrs = `{"days":90}`
				code.Code = req.Code
				ids, err := s.repo.CreateRedemptionCodes(context.Background(), []domain.RedemptionCode{
					code,
				})
				require.NoError(t, err)
				code.ID = ids[0]

				_, err = s.repo.SetUnusedRedemptionCodeStatusUsed(context.Background(), code.OwnerID, code.Code)
				require.NoError(t, err)

				return code
			},
			newEvtFunc: func(code domain.RedemptionCode) any {
				return event.MemberEvent{}
			},
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, eve any) mq.MQ {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)

				mockProducer := mocks.NewMockProducer(ctrl)

				mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller, q mq.MQ) *web.Handler {
				t.Helper()

				mockProductSvc := productmocks.NewMockService(ctrl)

				mockOrderSvc := ordermocks.NewMockService(ctrl)

				memberEventProducer, err := producer.NewMemberEventProducer(q)
				require.NoError(t, err)

				svc := service.NewService(s.repo, mockOrderSvc, mockProductSvc, nil, nil, memberEventProducer, nil, nil, nil)
				return web.NewHandler(svc)
			},

			after: func(t *testing.T, code domain.RedemptionCode) {
				t.Helper()
				c, err := s.repo.FindRedemptionCode(context.Background(), code.Code)
				require.NoError(t, err)
				require.Equal(t, code.Code, c.Code)
				require.Equal(t, domain.RedemptionCodeStatusUsed, c.Status)
			},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 412001,
				Msg:  "兑换码已使用",
			},
		},
		{
			name: "兑换失败_兑换码不正确",
			req: web.RedeemRedemptionCodeReq{
				Code: "redemption-code-all-004",
			},
			before: func(t *testing.T, req web.RedeemRedemptionCodeReq) domain.RedemptionCode {
				return domain.RedemptionCode{}
			},
			newEvtFunc: func(code domain.RedemptionCode) any {
				return event.MemberEvent{}
			},
			newMQFunc: func(t *testing.T, ctrl *gomock.Controller, eve any) mq.MQ {
				t.Helper()
				mockMQ := mocks.NewMockMQ(ctrl)

				mockProducer := mocks.NewMockProducer(ctrl)

				mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)
				return mockMQ
			},
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller, q mq.MQ) *web.Handler {
				t.Helper()

				mockProductSvc := productmocks.NewMockService(ctrl)

				mockOrderSvc := ordermocks.NewMockService(ctrl)

				memberEventProducer, err := producer.NewMemberEventProducer(q)
				require.NoError(t, err)

				svc := service.NewService(s.repo, mockOrderSvc, mockProductSvc, nil, nil, memberEventProducer, nil, nil, nil)
				return web.NewHandler(svc)
			},

			after:    func(t *testing.T, code domain.RedemptionCode) {},
			wantCode: 500,
			wantResp: test.Result[any]{
				Code: 412002,
				Msg:  "兑换码不正确",
			},
		},
		// 兑换失败 —— 超过限流次数1s一次
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/code/redeem", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			code := tc.before(t, tc.req)
			q := tc.newMQFunc(t, ctrl, tc.newEvtFunc(code))
			server := s.newGinServer(tc.newHandlerFunc(t, ctrl, q))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			require.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t, code)
		})
	}
}

func (s *ModuleTestSuite) TestHandler_ListRedemptionCode() {
	t := s.T()

	total := 100
	for idx := 0; idx < total; idx++ {
		id := int64(2000 + idx)
		status := domain.RedemptionCodeStatus(uint8(id)%2 + 1)
		code := s.newMemberRedemptionCodeDomain(testID, id, id)
		code.Status = status
		_, err := s.repo.CreateRedemptionCodes(context.Background(), []domain.RedemptionCode{code})
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

				redemptionCodeGenerator := s.getRedemptionCodeGenerator(sequencenumber.NewGenerator())
				svc := service.NewService(s.repo, nil, nil, redemptionCodeGenerator, nil, nil, nil, nil, nil)
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
							Code: "redemption-code-member-2099",
							Type: "member",
							SKU: web.SKU{
								SN:   fmt.Sprintf("sku-sn-%d", 2099),
								Name: fmt.Sprintf("sku-name-%d", 2099),
							},
							Status: domain.RedemptionCodeStatusUsed.ToUint8(),
						},
						{
							Code: "redemption-code-member-2098",
							Type: "member",
							SKU: web.SKU{
								SN:   fmt.Sprintf("sku-sn-%d", 2098),
								Name: fmt.Sprintf("sku-name-%d", 2098),
							},
							Status: domain.RedemptionCodeStatusUnused.ToUint8(),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
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

func (s *ModuleTestSuite) getRedemptionCodeGenerator(g *sequencenumber.Generator) func(id int64) string {
	redemptionCodeGenerator := func(generator *sequencenumber.Generator) func(id int64) string {
		return func(id int64) string {
			code, _ := generator.Generate(id)
			return code
		}
	}
	return redemptionCodeGenerator(g)
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

func (s *ModuleTestSuite) TestHandler_GenerateInvitationCode() {
	t := s.T()

	testCases := []struct {
		name string

		before         func(t *testing.T) repository.MarketingRepository
		newHandlerFunc func(t *testing.T, repo repository.MarketingRepository) *web.Handler
		wantCode       int
		after          func(t *testing.T, code string)
	}{
		{
			name: "首次生成邀请码",
			before: func(t *testing.T) repository.MarketingRepository {
				codeCache := cache.NewInvitationCodeECache(testioc.InitCache(), time.Minute*10)
				repo := repository.NewRepository(dao.NewGORMMarketingDAO(s.db), codeCache)
				return repo
			},
			newHandlerFunc: func(t *testing.T, repo repository.MarketingRepository) *web.Handler {
				t.Helper()
				codeGenerator := func(id int64) string {
					return fmt.Sprintf("invitation-code-1-%d", id)
				}
				svc := service.NewService(repo, nil, nil, codeGenerator, nil, nil, nil, nil, nil)
				return web.NewHandler(svc)
			},
			wantCode: 200,
			after: func(t *testing.T, code string) {
				t.Helper()
				require.Equal(t, fmt.Sprintf("invitation-code-1-%d", testID), code)
			},
		},
		{
			name: "一定时间内多次生成返回相同邀请码",
			before: func(t *testing.T) repository.MarketingRepository {
				t.Helper()

				codeCache := cache.NewInvitationCodeECache(testioc.InitCache(), time.Minute*10)
				repo := repository.NewRepository(dao.NewGORMMarketingDAO(s.db), codeCache)

				c := fmt.Sprintf("invitation-code-2-%d", testID)
				_, err := repo.CreateInvitationCode(context.Background(), domain.InvitationCode{
					Uid:  testID,
					Code: c,
				})
				require.NoError(t, err)

				return repo
			},
			newHandlerFunc: func(t *testing.T, repo repository.MarketingRepository) *web.Handler {
				t.Helper()
				codeGenerator := func(id int64) string {
					return fmt.Sprintf("invitation-code-3-%d", id)
				}
				svc := service.NewService(repo, nil, nil, codeGenerator, nil, nil, nil, nil, nil)
				return web.NewHandler(svc)
			},
			wantCode: 200,
			after: func(t *testing.T, code string) {
				t.Helper()
				require.Equal(t, fmt.Sprintf("invitation-code-2-%d", testID), code)
			},
		},
		{
			name: "一定时间后再次生成返回新的邀请码",
			before: func(t *testing.T) repository.MarketingRepository {
				t.Helper()

				duration := 10 * time.Millisecond
				codeCache := cache.NewInvitationCodeECache(testioc.InitCache(), duration)
				repo := repository.NewRepository(dao.NewGORMMarketingDAO(s.db), codeCache)

				c := fmt.Sprintf("invitation-code-4-%d", testID)
				_, err := repo.CreateInvitationCode(context.Background(), domain.InvitationCode{
					Uid:  testID,
					Code: c,
				})
				require.NoError(t, err)

				time.Sleep(time.Second)
				return repo
			},
			newHandlerFunc: func(t *testing.T, repo repository.MarketingRepository) *web.Handler {
				t.Helper()
				codeGenerator := func(id int64) string {
					return fmt.Sprintf("invitation-code-5-%d", id)
				}
				svc := service.NewService(repo, nil, nil, codeGenerator, nil, nil, nil, nil, nil)
				return web.NewHandler(svc)
			},
			wantCode: 200,
			after: func(t *testing.T, code string) {
				t.Helper()
				require.Equal(t, fmt.Sprintf("invitation-code-5-%d", testID), code)
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost,
				"/invitation/gen", nil)
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			repo := tc.before(t)
			server := s.newGinServer(tc.newHandlerFunc(t, repo))
			server.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantCode, recorder.Code)
			code, ok := recorder.MustScan().Data.(string)
			require.True(t, ok)
			tc.after(t, code)

			c := testioc.InitCache()
			_, err = c.Delete(context.Background(), fmt.Sprintf("marketing:invitation-code:user:%d", testID))
			require.NoError(t, err)
		})
	}
}

func (s *ModuleTestSuite) TestService_RedeemRedemptionCode() {
	t := s.T()

	oid := int64(101001)
	skuId := int64(4)
	code := s.newMemberRedemptionCodeDomain(testID, oid, skuId)
	ids, err := s.repo.CreateRedemptionCodes(context.Background(), []domain.RedemptionCode{
		code,
	})
	require.NoError(t, err)
	code.ID = ids[0]

	require.Equal(t, domain.RedemptionCodeStatusUnused, code.Status)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMQ := mocks.NewMockMQ(ctrl)

	mockProducer := mocks.NewMockProducer(ctrl)
	mockProducer.EXPECT().Produce(gomock.Any(), gomock.Any()).Return(&mq.ProducerResult{}, nil)
	mockMQ.EXPECT().Producer(event.MemberUpdateEventName).Return(mockProducer, nil)

	mockProductSvc := productmocks.NewMockService(ctrl)

	mockOrderSvc := ordermocks.NewMockService(ctrl)

	memberEventProducer, err := producer.NewMemberEventProducer(mockMQ)
	require.NoError(t, err)

	svc := service.NewService(s.repo, mockOrderSvc, mockProductSvc, nil, nil, memberEventProducer, nil, nil, nil)

	var wg sync.WaitGroup
	n := 100
	errChan := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errChan <- svc.RedeemRedemptionCode(context.Background(), int64(i), code.Code)
		}(i + 20001)
	}

	wg.Wait()

	close(errChan)
	errCounter := 0
	for e := range errChan {
		if e == nil {
			continue
		}
		require.ErrorIs(t, e, service.ErrRedemptionCodeUsed)
		errCounter++
	}
	require.Equal(t, n-1, errCounter)
	c, err := s.repo.FindRedemptionCode(context.Background(), code.Code)
	require.NoError(t, err)
	require.Equal(t, domain.RedemptionCodeStatusUsed, c.Status)
	require.NotEqual(t, c.Utime, c.Ctime)
}

func (s *ModuleTestSuite) newMemberRedemptionCodeDomain(ownerID int64, oid, skuId int64) domain.RedemptionCode {
	return domain.RedemptionCode{
		OwnerID: ownerID,
		Biz:     "order",
		BizId:   oid,
		Type:    "member",
		Attrs: domain.CodeAttrs{
			SKU: domain.SKU{
				ID:    skuId,
				SN:    fmt.Sprintf("sku-sn-%d", skuId),
				Name:  fmt.Sprintf("sku-name-%d", skuId),
				Attrs: `{"days":30}`},
		},
		Code:   fmt.Sprintf("redemption-code-member-%d", oid),
		Status: domain.RedemptionCodeStatusUnused,
	}
}

func (s *ModuleTestSuite) newProjectRedemptionCodeDomain(ownerID int64, oid, skuId int64) domain.RedemptionCode {
	return domain.RedemptionCode{
		OwnerID: ownerID,
		Biz:     "order",
		BizId:   oid,
		Type:    "project",
		Attrs: domain.CodeAttrs{
			SKU: domain.SKU{
				ID:   skuId,
				SN:   fmt.Sprintf("sku-sn-%d", skuId),
				Name: fmt.Sprintf("sku-name-%d", skuId),
			},
		},
		Code:   fmt.Sprintf("redemption-code-project-%d", oid),
		Status: domain.RedemptionCodeStatusUnused,
	}
}

func (s *ModuleTestSuite) TestAdminHandler_GenerateRedemptionCode() {
	t := s.T()

	testCases := []struct {
		name            string
		newAdminHandler func(t *testing.T, ctrl *gomock.Controller, req web.GenerateRedemptionCodeReq) *web.AdminHandler
		req             web.GenerateRedemptionCodeReq
		after           func(t *testing.T, req web.GenerateRedemptionCodeReq)

		wantCode int
		wantResp test.Result[any]
	}{
		{
			name: "生成多个兑换码",
			newAdminHandler: func(t *testing.T, ctrl *gomock.Controller, req web.GenerateRedemptionCodeReq) *web.AdminHandler {
				t.Helper()

				mockProductSvc := productmocks.NewMockService(ctrl)
				skuId := int64(30001)
				spuId := int64(30002)
				sku := product.SKU{
					ID:       skuId,
					SPUID:    spuId,
					SN:       "sku-sn-30001",
					Name:     fmt.Sprintf("sku-name-%d", skuId),
					Desc:     fmt.Sprintf("sku-desc-%d", skuId),
					Price:    1990,
					Stock:    9999,
					SaleType: product.SaleTypeUnlimited,
					Attrs:    fmt.Sprintf("sku-attrs-%d", skuId),
					Image:    fmt.Sprintf("sku-image-%d", skuId),
					Status:   product.StatusOnShelf,
				}
				mockProductSvc.EXPECT().FindSKUBySN(gomock.Any(), req.SKUSN).Return(sku, nil)
				spu := product.SPU{
					ID:        spuId,
					SN:        fmt.Sprintf("spu-sn-%d", spuId),
					Name:      fmt.Sprintf("spu-name-%d", spuId),
					Desc:      fmt.Sprintf("spu-desc-%d", spuId),
					Category0: fmt.Sprintf("spu-category0-%d", spuId),
					Category1: fmt.Sprintf("spu-category1-%d", spuId),
					SKUs:      []product.SKU{sku},
					Status:    product.StatusOnShelf,
				}
				mockProductSvc.EXPECT().FindSPUByID(gomock.Any(), sku.SPUID).Return(spu, nil)

				return web.NewAdminHandler(service.NewAdminService(s.repo), mockProductSvc, s.getRedemptionCodeGenerator(sequencenumber.NewGenerator()))
			},
			req: web.GenerateRedemptionCodeReq{
				Biz:   "admin",
				BizId: time.Now().UnixMilli(),
				SKUSN: "sku-sn-30001",
				Count: 3,
			},
			after: func(t *testing.T, req web.GenerateRedemptionCodeReq) {
				t.Helper()

				codes, err := s.repo.FindRedemptionCodesByUID(context.Background(), 0, 0, req.Count)
				require.NoError(t, err)
				skuId := int64(30001)
				spuId := int64(30002)
				code := domain.RedemptionCode{
					OwnerID: 0,
					Biz:     req.Biz,
					BizId:   req.BizId,
					Type:    fmt.Sprintf("spu-category1-%d", spuId),
					Attrs: domain.CodeAttrs{
						SKU: domain.SKU{
							ID:    skuId,
							SN:    fmt.Sprintf("sku-sn-%d", skuId),
							Name:  fmt.Sprintf("sku-name-%d", skuId),
							Attrs: fmt.Sprintf("sku-attrs-%d", skuId),
						},
					},
					Status: domain.RedemptionCodeStatusUnused,
				}
				expected := make([]domain.RedemptionCode, 0, req.Count)
				for i := 0; i < req.Count; i++ {
					expected = append(expected, code)
				}
				s.assertRedemptionCodeEqual(t, expected, codes)
			},
			wantCode: 200,
			wantResp: test.Result[any]{
				Msg: "OK",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/code/gen", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			server := s.newAdminGinServer(tc.newAdminHandler(t, ctrl, tc.req))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			require.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t, tc.req)
		})

	}

}

func (s *ModuleTestSuite) TestAdminHandler_ListRedemptionCode() {
	t := s.T()

	total := 100
	for idx := 0; idx < total; idx++ {
		id := int64(3000 + idx)
		status := domain.RedemptionCodeStatus(uint8(id)%2 + 1)
		code := s.newProjectRedemptionCodeDomain(0, id, id)
		code.Status = status
		_, err := s.repo.CreateRedemptionCodes(context.Background(), []domain.RedemptionCode{code})
		require.NoError(t, err)
	}

	testCases := []struct {
		name           string
		newHandlerFunc func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler
		req            web.ListRedemptionCodesReq

		wantCode int
		wantResp test.Result[web.ListRedemptionCodesResp]
	}{
		{
			name: "获取成功",
			newHandlerFunc: func(t *testing.T, ctrl *gomock.Controller) *web.AdminHandler {
				t.Helper()

				redemptionCodeGenerator := s.getRedemptionCodeGenerator(sequencenumber.NewGenerator())
				svc := service.NewAdminService(s.repo)
				return web.NewAdminHandler(svc, nil, redemptionCodeGenerator)
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
							Code: "redemption-code-project-3099",
							Type: "project",
							SKU: web.SKU{
								SN:   fmt.Sprintf("sku-sn-%d", 3099),
								Name: fmt.Sprintf("sku-name-%d", 3099),
							},
							Status: domain.RedemptionCodeStatusUsed.ToUint8(),
						},
						{
							Code: "redemption-code-project-3098",
							Type: "project",
							SKU: web.SKU{
								SN:   fmt.Sprintf("sku-sn-%d", 3098),
								Name: fmt.Sprintf("sku-name-%d", 3098),
							},
							Status: domain.RedemptionCodeStatusUnused.ToUint8(),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			req, err := http.NewRequest(http.MethodPost,
				"/code/list", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.ListRedemptionCodesResp]()
			server := s.newAdminGinServer(tc.newHandlerFunc(t, ctrl))
			server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			s.assertListRedemptionCodesRespEqual(t, tc.wantResp.Data, recorder.MustScan().Data)
		})
	}
}
