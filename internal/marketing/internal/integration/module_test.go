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
	"testing"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/marketing/internal/event"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/consumer"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/marketing/internal/service"
	"github.com/ecodeclub/webook/internal/order"
	ordermocks "github.com/ecodeclub/webook/internal/order/mocks"
	"github.com/ecodeclub/webook/internal/test/mocks"
	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestMarketingModule(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

type ModuleTestSuite struct {
	suite.Suite
	db *egorm.Component
}

func (s *ModuleTestSuite) SetupSuite() {

}

func (s *ModuleTestSuite) TearDownSuite() {

}

func (s *ModuleTestSuite) TearDownTest() {

}

func (s *ModuleTestSuite) TestConsumer_ConsumeOrderEvent() {
	t := s.T()

	testCases := []struct {
		name       string
		newMQFunc  func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent) mq.MQ
		newSvcFunc func(t *testing.T, ctrl *gomock.Controller, evt event.OrderEvent, q mq.MQ) service.Service
		evt        event.OrderEvent
		after      func(t *testing.T)

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

				return service.NewService(mockOrderSvc, memberEventProducer)
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
				return service.NewService(mockOrderSvc, memberEventProducer)
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
