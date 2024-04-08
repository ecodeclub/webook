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
	"testing"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit/internal/domain"
	"github.com/ecodeclub/webook/internal/credit/internal/event"
	"github.com/ecodeclub/webook/internal/credit/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/credit/internal/service"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestCreditModule(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

type ModuleTestSuite struct {
	suite.Suite
	db    *egorm.Component
	mq    mq.MQ
	cache ecache.Cache
	svc   service.Service
}

func (s *ModuleTestSuite) SetupTest() {
	s.svc = startup.InitService()
	s.mq = testioc.InitMQ()
	s.db = testioc.InitDB()
	s.cache = testioc.InitCache()
}

func (s *ModuleTestSuite) TearDownSuite() {
	// err := s.db.Exec("DROP TABLE `credits`").Error
	// s.NoError(err)
	// err = s.db.Exec("DROP TABLE `credit_logs`").Error
	// s.NoError(err)
}

func (s *ModuleTestSuite) TearDownTest() {
	// err := s.db.Exec("TRUNCATE TABLE `credits`").Error
	// s.NoError(err)
	// err = s.db.Exec("TRUNCATE TABLE `credit_logs`").Error
	// s.NoError(err)
}

func (s *ModuleTestSuite) TestConsumer_ConsumeCreditIncreaseEvent() {
	// 单消费者顺序消费
	t := s.T()

	producer, er := s.mq.Producer("credit_increase_events")
	require.NoError(t, er)

	consumer, er := event.NewCreditIncreaseConsumer(s.svc, s.mq, s.cache)
	require.NoError(t, er)
	t.Cleanup(func() {
		require.NoError(t, consumer.Stop(context.Background()))
	})

	testCases := []struct {
		name string

		before func(t *testing.T, producer mq.Producer, message *mq.Message)
		after  func(t *testing.T, evt event.CreditIncreaseEvent)
		evt    event.CreditIncreaseEvent

		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "增加积分成功_新增用户",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, evt event.CreditIncreaseEvent) {
				key := fmt.Sprintf("webook:credit:increase:%s", evt.Key)
				t.Logf("after evt.Key=%#v, key = %#v\n", evt.Key, key)
				_, err := s.cache.Delete(context.Background(), key)
				require.NoError(t, err)

				uid := int64(6001)

				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Len(t, c.Logs, 1)
				require.Equal(t, int64(domain.CreditLogStatusActive), c.Logs[0].Status)

				require.Equal(t, evt, event.CreditIncreaseEvent{
					Key:     evt.Key,
					Uid:     c.Uid,
					Amount:  c.TotalAmount,
					BizId:   c.Logs[0].BizId,
					BizType: c.Logs[0].BizType,
					Action:  c.Logs[0].Action,
				})
			},
			evt: event.CreditIncreaseEvent{
				Key:     "sn-key-6001",
				Uid:     6001,
				Amount:  100,
				BizId:   1,
				BizType: 1,
				Action:  "注册",
			},
			errAssertFunc: assert.NoError,
		},
		{
			name: "增加积分成功_已有用户_无预扣积分",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {

				// 创建已有用户
				uid := int64(6002)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 100,
					Logs: []domain.CreditLog{
						{
							BizId:   2,
							BizType: 2,
							Action:  "邀请注册",
						},
					},
				})
				require.NoError(t, err)

				// 发送消息
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)

				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, evt event.CreditIncreaseEvent) {
				key := fmt.Sprintf("webook:credit:increase:%s", evt.Key)
				t.Logf("after evt.Key=%#v, key = %#v\n", evt.Key, key)
				_, err := s.cache.Delete(context.Background(), key)
				require.NoError(t, err)

				uid := int64(6002)
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Len(t, c.Logs, 2)
				require.Equal(t, int64(domain.CreditLogStatusActive), c.Logs[0].Status)
				changedAmount := c.TotalAmount - int64(100)
				require.Equal(t, evt, event.CreditIncreaseEvent{
					Key:     evt.Key,
					Uid:     c.Uid,
					Amount:  changedAmount,
					BizId:   c.Logs[0].BizId,
					BizType: c.Logs[0].BizType,
					Action:  c.Logs[0].Action,
				})
			},
			evt: event.CreditIncreaseEvent{
				Key:     "sn-key-6002",
				Uid:     6002,
				Amount:  250,
				BizId:   3,
				BizType: 3,
				Action:  "购买商品",
			},
			errAssertFunc: assert.NoError,
		},
		// todo: 增加积分成功_已有用户_有预扣积分
		// todo: changeAmount <= 0
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			message := s.newCreditIncreaseEventMessage(t, tc.evt)
			tc.before(t, producer, message)

			err := consumer.Consume(context.Background())
			// 模拟重复消费
			err = consumer.Consume(context.Background())

			tc.errAssertFunc(t, err)
			tc.after(t, tc.evt)
			t.Logf("test end\n")
		})
	}
}

func (s *ModuleTestSuite) newCreditIncreaseEventMessage(t *testing.T, evt event.CreditIncreaseEvent) *mq.Message {
	t.Helper()
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}
