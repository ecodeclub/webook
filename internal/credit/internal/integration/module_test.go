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
	"github.com/ecodeclub/webook/internal/credit/internal/domain"
	"github.com/ecodeclub/webook/internal/credit/internal/event"
	"github.com/ecodeclub/webook/internal/credit/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/credit/internal/service"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestCreditModule(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

type ModuleTestSuite struct {
	suite.Suite
	db  *egorm.Component
	mq  mq.MQ
	svc service.Service
}

func (s *ModuleTestSuite) SetupTest() {
	s.svc = startup.InitService()
	s.mq = testioc.InitMQ()
	s.db = testioc.InitDB()
}

func (s *ModuleTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `credits`").Error
	s.NoError(err)
	err = s.db.Exec("DROP TABLE `credit_logs`").Error
	s.NoError(err)
}

func (s *ModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `credits`").Error
	s.NoError(err)
	err = s.db.Exec("TRUNCATE TABLE `credit_logs`").Error
	s.NoError(err)
}

func (s *ModuleTestSuite) TestConsumer_ConsumeCreditIncreaseEvent() {
	// 单消费者顺序消费
	t := s.T()

	producer, er := s.mq.Producer("credit_increase_events")
	require.NoError(t, er)

	consumer, er := event.NewCreditIncreaseConsumer(s.svc, s.mq)
	require.NoError(t, er)
	t.Cleanup(func() {
		require.NoError(t, consumer.Stop(context.Background()))
	})

	testCases := []struct {
		name string

		before func(t *testing.T, producer mq.Producer, message *mq.Message)
		after  func(t *testing.T, evt event.CreditIncreaseEvent)
		evt    event.CreditIncreaseEvent

		errAssertFunc require.ErrorAssertionFunc
	}{
		{
			name: "增加积分成功_新增用户",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()

				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, evt event.CreditIncreaseEvent) {
				t.Helper()

				uid := int64(6001)

				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Len(t, c.Logs, 1)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:    "key-6001",
						BizId:  1,
						Biz:    1,
						Action: "注册",
					},
				})
			},
			evt: event.CreditIncreaseEvent{
				Key:    "key-6001",
				Uid:    6001,
				Amount: 100,
				BizId:  1,
				Biz:    1,
				Action: "注册",
			},
			errAssertFunc: require.NoError,
		},
		{
			name: "增加积分成功_已有用户_无预扣积分",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()

				// 创建已有用户
				uid := int64(6002)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 100,
					Logs: []domain.CreditLog{
						{
							Key:    "key-6002-1",
							BizId:  2,
							Biz:    2,
							Action: "邀请注册",
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
				t.Helper()

				uid := int64(6002)
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Equal(t, uint64(350), c.TotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:    "key-6002-2",
						BizId:  3,
						Biz:    3,
						Action: "购买商品",
					},
					{
						Key:    "key-6002-1",
						BizId:  2,
						Biz:    2,
						Action: "邀请注册",
					},
				})
			},
			evt: event.CreditIncreaseEvent{
				Key:    "key-6002-2",
				Uid:    6002,
				Amount: 250,
				BizId:  3,
				Biz:    3,
				Action: "购买商品",
			},
			errAssertFunc: require.NoError,
		},
		// todo: 增加积分成功_已有用户_有预扣积分
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
		})
	}
}

func (s *ModuleTestSuite) newCreditIncreaseEventMessage(t *testing.T, evt event.CreditIncreaseEvent) *mq.Message {
	t.Helper()
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) TestService_TryDeductCredits() {

	t := s.T()

	testCases := []struct {
		name string

		before        func(t *testing.T)
		after         func(t *testing.T)
		credit        domain.Credit
		errAssertFunc require.ErrorAssertionFunc
	}{
		{
			name: "预扣积分成功_用户积分充足_有剩余",
			before: func(t *testing.T) {
				t.Helper()

				// 创建已有用户
				uid := int64(7001)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 100,
					Logs: []domain.CreditLog{
						{
							Key:    "key-7001-1",
							BizId:  2,
							Biz:    2,
							Action: "邀请注册",
						},
					},
				})
				require.NoError(t, err)
			},
			credit: domain.Credit{
				Uid:          7001,
				ChangeAmount: 70,
				Logs: []domain.CreditLog{
					{
						Key:    "key-7001-2",
						BizId:  7,
						Biz:    7,
						Action: "购买商品",
					},
				},
			},
			after: func(t *testing.T) {
				t.Helper()

				uid := int64(7001)
				expectedTotalAmount := uint64(30)
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Equal(t, uid, c.Uid)
				require.Equal(t, expectedTotalAmount, c.TotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:    "key-7001-1",
						BizId:  2,
						Biz:    2,
						Action: "邀请注册",
					},
				})
			},
			errAssertFunc: require.NoError,
		},
		{
			name: "预扣积分成功_用户积分充足_归为零",
			before: func(t *testing.T) {
				t.Helper()

				// 创建已有用户
				uid := int64(7002)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 100,
					Logs: []domain.CreditLog{
						{
							Key:    "key-7002-1",
							BizId:  2,
							Biz:    2,
							Action: "首次注册",
						},
					},
				})
				require.NoError(t, err)
			},
			credit: domain.Credit{
				Uid:          7002,
				ChangeAmount: 100,
				Logs: []domain.CreditLog{
					{
						Key:    "key-7002-2",
						BizId:  7,
						Biz:    7,
						Action: "购买项目",
					},
				},
			},
			after: func(t *testing.T) {
				t.Helper()

				uid := int64(7002)
				expectedTotalAmount := uint64(0)
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Equal(t, uid, c.Uid)
				require.Equal(t, expectedTotalAmount, c.TotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:    "key-7002-1",
						BizId:  2,
						Biz:    2,
						Action: "首次注册",
					},
				})
			},
			errAssertFunc: require.NoError,
		},
		{
			name: "预扣积分失败_用户积分不足",
			before: func(t *testing.T) {
				t.Helper()

				// 创建已有用户
				uid := int64(7003)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 100,
					Logs: []domain.CreditLog{
						{
							Key:    "key-7003-1",
							BizId:  4,
							Biz:    4,
							Action: "首次注册",
						},
					},
				})
				require.NoError(t, err)
			},
			credit: domain.Credit{
				Uid:          7003,
				ChangeAmount: 101,
				Logs: []domain.CreditLog{
					{
						Key:    "key-7003-2",
						BizId:  8,
						Biz:    8,
						Action: "购买专栏",
					},
				},
			},
			after: func(t *testing.T) {
				t.Helper()

				uid := int64(7003)
				expectedTotalAmount := uint64(100)
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Equal(t, uid, c.Uid)
				require.Equal(t, expectedTotalAmount, c.TotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:    "key-7003-1",
						BizId:  4,
						Biz:    4,
						Action: "首次注册",
					},
				})
			},
			errAssertFunc: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorIs(t, err, service.ErrCreditNotEnough)
			},
		},
		{
			name:   "预扣积分失败_用户无记录",
			before: func(t *testing.T) {},
			credit: domain.Credit{
				Uid:          7004,
				ChangeAmount: 10,
				Logs: []domain.CreditLog{
					{
						Key:    "key-7004-1",
						BizId:  9,
						Biz:    9,
						Action: "购买专栏",
					},
				},
			},
			after:         func(t *testing.T) {},
			errAssertFunc: require.Error,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			id, err := s.svc.TryDeductCredits(context.Background(), tc.credit)
			tc.errAssertFunc(t, err)
			if err == nil {
				require.NotZero(t, id)
			}
			tc.after(t)
		})
	}
}

func (s *ModuleTestSuite) TestService_ConfirmDeductCredits() {
	t := s.T()

	testCases := []struct {
		name          string
		getUIDAndTID  func(t *testing.T) (int64, int64)
		after         func(t *testing.T)
		errAssertFunc require.ErrorAssertionFunc
	}{
		{
			name: "确认预扣成功_ID有效",
			getUIDAndTID: func(t *testing.T) (int64, int64) {
				t.Helper()
				// 创建已有用户
				uid := int64(8001)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 100,
					Logs: []domain.CreditLog{
						{
							Key:    "key-8001-1",
							BizId:  1,
							Biz:    1,
							Action: "注册",
						},
					},
				})
				require.NoError(t, err)
				// 预扣
				id, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 50,
					Logs: []domain.CreditLog{
						{
							Key:    "key-8001-2",
							BizId:  9,
							Biz:    9,
							Action: "购买面试",
						},
					},
				})
				require.NoError(t, err)
				return uid, id
			},
			after: func(t *testing.T) {
				t.Helper()

				uid := int64(8001)
				expectedTotalAmount := uint64(50)
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Equal(t, uid, c.Uid)
				require.Equal(t, expectedTotalAmount, c.TotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:    "key-8001-2",
						BizId:  9,
						Biz:    9,
						Action: "购买面试",
					},
					{
						Key:    "key-8001-1",
						BizId:  1,
						Biz:    1,
						Action: "注册",
					},
				})
			},
			errAssertFunc: require.NoError,
		},
		{
			name: "确认预扣失败_ID有效但非法",
			getUIDAndTID: func(t *testing.T) (int64, int64) {
				t.Helper()
				// 创建已有用户
				uid := int64(8002)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 100,
					Logs: []domain.CreditLog{
						{
							Key:    "key-8002-1",
							BizId:  1,
							Biz:    1,
							Action: "注册",
						},
					},
				})
				require.NoError(t, err)
				return uid, int64(1)
			},
			after: func(t *testing.T) {
				t.Helper()

				uid := int64(8002)
				expectedTotalAmount := uint64(100)
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Equal(t, uid, c.Uid)
				require.Equal(t, expectedTotalAmount, c.TotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:    "key-8002-1",
						BizId:  1,
						Biz:    1,
						Action: "注册",
					},
				})
			},
			errAssertFunc: require.Error,
		},
		{
			name:          "确认预扣失败_ID非法",
			getUIDAndTID:  func(t *testing.T) (int64, int64) { return int64(8002), int64(1000) },
			after:         func(t *testing.T) {},
			errAssertFunc: require.Error,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			uid, tid := tc.getUIDAndTID(t)
			err := s.svc.ConfirmDeductCredits(context.Background(), uid, tid)
			tc.errAssertFunc(t, err)
			tc.after(t)
		})
	}
}

func (s *ModuleTestSuite) TestService_CancelDeductCredits() {
	t := s.T()

	testCases := []struct {
		name          string
		getUIDAndTID  func(t *testing.T) (int64, int64)
		after         func(t *testing.T)
		errAssertFunc require.ErrorAssertionFunc
	}{
		{
			name: "取消预扣成功_ID有效",
			getUIDAndTID: func(t *testing.T) (int64, int64) {
				t.Helper()
				// 创建已有用户
				uid := int64(9001)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 100,
					Logs: []domain.CreditLog{
						{
							Key:    "key-9001-1",
							BizId:  1,
							Biz:    1,
							Action: "注册",
						},
					},
				})
				require.NoError(t, err)
				// 预扣
				tid, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 50,
					Logs: []domain.CreditLog{
						{
							Key:    "key-9001-2",
							BizId:  9,
							Biz:    9,
							Action: "购买面试",
						},
					},
				})
				require.NoError(t, err)
				return uid, tid
			},
			after: func(t *testing.T) {
				t.Helper()

				uid := int64(9001)
				expectedTotalAmount := uint64(100)
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Equal(t, uid, c.Uid)
				require.Equal(t, expectedTotalAmount, c.TotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:    "key-9001-1",
						BizId:  1,
						Biz:    1,
						Action: "注册",
					},
				})
			},
			errAssertFunc: require.NoError,
		},
		{
			name: "取消预扣失败_ID有效但非法",
			getUIDAndTID: func(t *testing.T) (int64, int64) {
				t.Helper()
				// 创建已有用户
				uid := int64(9002)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid:          uid,
					ChangeAmount: 100,
					Logs: []domain.CreditLog{
						{
							Key:    "key-9002-1",
							BizId:  1,
							Biz:    1,
							Action: "注册",
						},
					},
				})
				require.NoError(t, err)
				return uid, int64(1)
			},
			after: func(t *testing.T) {
				t.Helper()

				uid := int64(9002)
				expectedTotalAmount := uint64(100)
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Equal(t, uid, c.Uid)
				require.Equal(t, expectedTotalAmount, c.TotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:    "key-9002-1",
						BizId:  1,
						Biz:    1,
						Action: "注册",
					},
				})
			},
			errAssertFunc: require.Error,
		},
		{
			name:          "取消预扣失败_ID非法",
			getUIDAndTID:  func(t *testing.T) (int64, int64) { return int64(9002), int64(2000) },
			after:         func(t *testing.T) {},
			errAssertFunc: require.Error,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			uid, tid := tc.getUIDAndTID(t)
			err := s.svc.CancelDeductCredits(context.Background(), uid, tid)
			tc.errAssertFunc(t, err)
			tc.after(t)
		})
	}
}
