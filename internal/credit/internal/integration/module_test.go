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

	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit/internal/domain"
	"github.com/ecodeclub/webook/internal/credit/internal/event"
	"github.com/ecodeclub/webook/internal/credit/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/credit/internal/service"
	"github.com/ecodeclub/webook/internal/credit/internal/web"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const testUID = int64(230919)

func TestCreditModule(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

type ModuleTestSuite struct {
	suite.Suite
	server *egin.Component
	db     *egorm.Component
	mq     mq.MQ
	svc    service.Service
}

func (s *ModuleTestSuite) SetupTest() {
	s.svc = startup.InitService()

	handler := startup.InitHandler(s.svc)
	econf.Set("server", map[string]any{"contextTimeout": "1s"})
	server := egin.Load("server").Build()
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: testUID,
		}))
	})
	handler.PrivateRoutes(server.Engine)

	s.server = server
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

		errRequireFunc require.ErrorAssertionFunc
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

				require.Equal(t, uint64(100), c.TotalAmount)
				require.Equal(t, uint64(0), c.LockedTotalAmount)
				require.Len(t, c.Logs, 1)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-6001",
						ChangeAmount: 100,
						Biz:          "user",
						BizId:        1,
						Desc:         "注册",
					},
				})
			},
			evt: event.CreditIncreaseEvent{
				Key:    "key-6001",
				Uid:    6001,
				Amount: 100,
				Biz:    "user",
				BizId:  1,
				Action: "注册",
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "增加积分成功_已有用户_无预扣积分",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()

				// 创建已有用户
				uid := int64(6002)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-6002-1",
							ChangeAmount: 100,
							Biz:          "Marketing",
							BizId:        2,
							Desc:         "邀请注册",
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
				require.Equal(t, uint64(0), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-6002-2",
						ChangeAmount: 250,
						Biz:          "order",
						BizId:        3,
						Desc:         "购买商品",
					},
					{
						Key:          "key-6002-1",
						ChangeAmount: 100,
						Biz:          "Marketing",
						BizId:        2,
						Desc:         "邀请注册",
					},
				})
			},
			evt: event.CreditIncreaseEvent{
				Key:    "key-6002-2",
				Uid:    6002,
				Amount: 250,
				Biz:    "order",
				BizId:  3,
				Action: "购买商品",
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "增加积分成功_已有用户_有预扣积分",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()

				// 创建已有用户
				uid := int64(6003)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-6003-1",
							ChangeAmount: 100,
							Biz:          "Marketing",
							BizId:        2,
							Desc:         "邀请注册",
						},
					},
				})
				require.NoError(t, err)

				// 预扣
				id, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-6003-2",
							ChangeAmount: 50,
							Biz:          "order",
							BizId:        9,
							Desc:         "购买面试",
						},
					},
				})
				require.NoError(t, err)
				require.NotZero(t, id)

				// 发送消息
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)

				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, evt event.CreditIncreaseEvent) {
				t.Helper()

				uid := int64(6003)
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)

				require.Equal(t, uint64(250), c.TotalAmount)
				require.Equal(t, uint64(50), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-6003-3",
						ChangeAmount: 200,
						Biz:          "order",
						BizId:        3,
						Desc:         "购买商品",
					},
					{
						Key:          "key-6003-2",
						ChangeAmount: -50,
						Biz:          "order",
						BizId:        9,
						Desc:         "购买面试",
					},
					{
						Key:          "key-6003-1",
						ChangeAmount: 100,
						Biz:          "Marketing",
						BizId:        2,
						Desc:         "邀请注册",
					},
				})
			},
			evt: event.CreditIncreaseEvent{
				Key:    "key-6003-3",
				Uid:    6003,
				Amount: 200,
				Biz:    "order",
				BizId:  3,
				Action: "购买商品",
			},
			errRequireFunc: require.NoError,
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {

			message := s.newCreditIncreaseEventMessage(t, tc.evt)
			tc.before(t, producer, message)

			err := consumer.Consume(context.Background())
			tc.errRequireFunc(t, err)

			// 模拟重复消费
			err = consumer.Consume(context.Background())
			tc.errRequireFunc(t, err)

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

func (s *ModuleTestSuite) TestService_AddCredits_Concurrent() {
	t := s.T()

	t.Run("相同消息并发创建_只有一个能够成功", func(t *testing.T) {
		n := 10

		uid := int64(3100)
		waitChan := make(chan struct{})
		errChan := make(chan error)

		for i := 0; i < n; i++ {
			go func() {
				<-waitChan
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "Key-Same-Message",
							ChangeAmount: 100,
							Biz:          "相同消息并发测试",
							BizId:        11,
							Desc:         "相同消息并发测试",
						},
					},
				})
				errChan <- err
			}()
		}

		time.Sleep(100 * time.Millisecond)
		close(waitChan)
		errCounter := 0
		for i := 0; i < n; i++ {
			err := <-errChan
			if err != nil {
				require.ErrorIs(t, err, service.ErrDuplicatedCreditLog)
				errCounter++
			}
		}
		require.Equal(t, n-1, errCounter)
		c, err := s.svc.GetCreditsByUID(context.Background(), uid)
		require.NoError(t, err)
		require.Equal(t, uint64(100), c.TotalAmount)
		require.Equal(t, uint64(0), c.LockedTotalAmount)
		require.Len(t, c.Logs, 1)
	})

	t.Run("不同消息并发创建或更新_均能成功", func(t *testing.T) {
		n := 10
		uid := int64(3200)
		changeAmount := 200
		waitChan := make(chan struct{})
		errChan := make(chan error)
		for i := 0; i < n; i++ {
			go func(i int) {
				<-waitChan
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          fmt.Sprintf("Key-diff-Message-%d", i),
							ChangeAmount: int64(changeAmount),
							Biz:          "不同消息并发测试",
							BizId:        12,
							Desc:         "不同消息并发测试",
						},
					},
				})
				errChan <- err
			}(i)
		}

		time.Sleep(100 * time.Millisecond)
		close(waitChan)
		for i := 0; i < n; i++ {
			require.NoError(t, <-errChan)
		}
		c, err := s.svc.GetCreditsByUID(context.Background(), uid)
		require.NoError(t, err)
		require.Equal(t, uint64(n*changeAmount), c.TotalAmount)
		require.Equal(t, uint64(0), c.LockedTotalAmount)
		require.Len(t, c.Logs, n)
	})

}

func (s *ModuleTestSuite) TestService_TryDeductCredits() {
	t := s.T()
	testCases := []struct {
		name string

		before         func(t *testing.T, uid int64)
		after          func(t *testing.T, uid int64)
		credit         domain.Credit
		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name: "预扣积分成功_用户积分充足_有剩余",
			before: func(t *testing.T, uid int64) {
				t.Helper()

				// 创建已有用户
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-7001-1",
							ChangeAmount: 100,
							Biz:          "Marketing",
							BizId:        2,
							Desc:         "邀请注册",
						},
					},
				})
				require.NoError(t, err)
			},
			credit: domain.Credit{
				Uid: 7001,
				Logs: []domain.CreditLog{
					{
						Key:          "key-7001-2",
						ChangeAmount: 70,
						Biz:          "order",
						BizId:        7,
						Desc:         "购买商品",
					},
				},
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, uint64(30), c.TotalAmount)
				require.Equal(t, uint64(70), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-7001-2",
						ChangeAmount: -70,
						Biz:          "order",
						BizId:        7,
						Desc:         "购买商品",
					},
					{
						Key:          "key-7001-1",
						ChangeAmount: 100,
						Biz:          "Marketing",
						BizId:        2,
						Desc:         "邀请注册",
					},
				})
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "预扣积分成功_用户积分充足_归为零",
			before: func(t *testing.T, uid int64) {
				t.Helper()

				// 创建已有用户
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-7002-1",
							ChangeAmount: 100,
							Biz:          "Marketing",
							BizId:        2,
							Desc:         "首次注册",
						},
					},
				})
				require.NoError(t, err)
			},
			credit: domain.Credit{
				Uid: 7002,
				Logs: []domain.CreditLog{
					{
						Key:          "key-7002-2",
						ChangeAmount: 100,
						Biz:          "order",
						BizId:        7,
						Desc:         "购买项目",
					},
				},
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, uint64(0), c.TotalAmount)
				require.Equal(t, uint64(100), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-7002-2",
						ChangeAmount: -100,
						Biz:          "order",
						BizId:        7,
						Desc:         "购买项目",
					},
					{
						Key:          "key-7002-1",
						Biz:          "Marketing",
						BizId:        2,
						Desc:         "首次注册",
						ChangeAmount: 100,
					},
				})
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "预扣积分失败_用户积分不足",
			before: func(t *testing.T, uid int64) {
				t.Helper()

				// 创建已有用户
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-7003-1",
							ChangeAmount: 100,
							Biz:          "user",
							BizId:        4,
							Desc:         "首次注册",
						},
					},
				})
				require.NoError(t, err)
			},
			credit: domain.Credit{
				Uid: 7003,
				Logs: []domain.CreditLog{
					{
						Key:          "key-7003-2",
						ChangeAmount: 101,
						Biz:          "order",
						BizId:        8,
						Desc:         "购买专栏",
					},
				},
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, uint64(100), c.TotalAmount)
				require.Equal(t, uint64(0), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-7003-1",
						ChangeAmount: 100,
						Biz:          "user",
						BizId:        4,
						Desc:         "首次注册",
					},
				})
			},
			errRequireFunc: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorIs(t, err, service.ErrCreditNotEnough)
			},
		},
		{
			name:   "预扣积分失败_用户无记录",
			before: func(t *testing.T, uid int64) {},
			credit: domain.Credit{
				Uid: 7004,
				Logs: []domain.CreditLog{
					{
						Key:          "key-7004-1",
						ChangeAmount: 10,
						Biz:          "order",
						BizId:        9,
						Desc:         "购买专栏",
					},
				},
			},
			after:          func(t *testing.T, uid int64) {},
			errRequireFunc: require.Error,
		},
		{
			name:   "预扣积分失败_积分流水记录非法",
			before: func(t *testing.T, uid int64) {},
			credit: domain.Credit{
				Uid:  7005,
				Logs: []domain.CreditLog{},
			},
			after: func(t *testing.T, uid int64) {},
			errRequireFunc: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorIs(t, err, service.ErrInvalidCreditLog)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t, tc.credit.Uid)
			id, err := s.svc.TryDeductCredits(context.Background(), tc.credit)
			tc.errRequireFunc(t, err)
			if err == nil {
				require.NotZero(t, id)
			}
			tc.after(t, tc.credit.Uid)
		})
	}
}

func (s *ModuleTestSuite) TestService_TryDeductCredits_Concurrent() {

	t := s.T()

	type Result struct {
		ID  int64
		Err error
	}

	t.Run("相同请求_支持扣减多次_执行多次返回第一次成功的结果", func(t *testing.T) {

		// 创建用户记录
		uid := int64(19001)
		err := s.svc.AddCredits(context.Background(), domain.Credit{
			Uid: uid,
			Logs: []domain.CreditLog{
				{
					Key:          "key-19001-1",
					ChangeAmount: 100,
					Biz:          "user",
					BizId:        19001,
					Desc:         "首次注册",
				},
			},
		})
		require.NoError(t, err)

		n := 20

		waitChan := make(chan struct{})
		resChan := make(chan Result)

		for i := 0; i < n; i++ {
			go func() {
				<-waitChan
				id, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-19001-2",
							ChangeAmount: 50,
							Biz:          "order",
							BizId:        9,
							Desc:         "购买专栏",
						},
					},
				})
				resChan <- Result{ID: id, Err: err}
			}()
		}

		time.Sleep(100 * time.Millisecond)
		close(waitChan)
		var expectedID int64
		for i := 0; i < n; i++ {
			res := <-resChan
			require.NoError(t, res.Err)
			require.NotZero(t, res.ID)
			if expectedID == 0 {
				expectedID = res.ID
			}
			require.Equal(t, expectedID, res.ID)
		}
		c, err := s.svc.GetCreditsByUID(context.Background(), uid)
		require.NoError(t, err)
		assert.Equal(t, uint64(50), c.TotalAmount)
		assert.Equal(t, uint64(50), c.LockedTotalAmount)
		assert.Equal(t, c.Logs, []domain.CreditLog{
			{
				Key:          "key-19001-2",
				ChangeAmount: -50,
				Biz:          "order",
				BizId:        9,
				Desc:         "购买专栏",
			},
			{
				Key:          "key-19001-1",
				ChangeAmount: 100,
				Biz:          "user",
				BizId:        19001,
				Desc:         "首次注册",
			},
		})
	})

	t.Run("相同请求_支持扣减一次_执行多次返回第一次成功的结果", func(t *testing.T) {

		// 创建用户记录
		uid := int64(19004)
		err := s.svc.AddCredits(context.Background(), domain.Credit{
			Uid: uid,
			Logs: []domain.CreditLog{
				{
					Key:          "key-19004-1",
					ChangeAmount: 50,
					Biz:          "user",
					BizId:        uid,
					Desc:         "首次注册",
				},
			},
		})
		require.NoError(t, err)

		n := 20

		waitChan := make(chan struct{})
		resChan := make(chan Result)

		for i := 0; i < n; i++ {
			go func() {
				<-waitChan
				id, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-19004-2",
							ChangeAmount: 50,
							Biz:          "order",
							BizId:        9,
							Desc:         "购买专栏",
						},
					},
				})
				resChan <- Result{ID: id, Err: err}
			}()
		}

		time.Sleep(100 * time.Millisecond)
		close(waitChan)
		var expectedID int64
		for i := 0; i < n; i++ {
			res := <-resChan
			require.NoError(t, res.Err)
			require.NotZero(t, res.ID)
			if expectedID == 0 {
				expectedID = res.ID
			}
			require.Equal(t, expectedID, res.ID)
		}
		c, err := s.svc.GetCreditsByUID(context.Background(), uid)
		require.NoError(t, err)
		require.Equal(t, uint64(0), c.TotalAmount)
		require.Equal(t, uint64(50), c.LockedTotalAmount)
		require.Equal(t, c.Logs, []domain.CreditLog{
			{
				Key:          "key-19004-2",
				ChangeAmount: -50,
				Biz:          "order",
				BizId:        9,
				Desc:         "购买专栏",
			},
			{
				Key:          "key-19004-1",
				ChangeAmount: 50,
				Biz:          "user",
				BizId:        uid,
				Desc:         "首次注册",
			},
		})
	})

	t.Run("不同请求_部分成功", func(t *testing.T) {

		// 创建用户记录
		uid := int64(19002)
		er := s.svc.AddCredits(context.Background(), domain.Credit{
			Uid: uid,
			Logs: []domain.CreditLog{
				{
					Key:          "key-19002",
					ChangeAmount: 100,
					Biz:          "user",
					BizId:        19002,
					Desc:         "首次注册",
				},
			},
		})
		require.NoError(t, er)

		n := 10

		waitChan := make(chan struct{})
		resChan := make(chan Result)
		logChan := make(chan domain.CreditLog, n)

		for i := 0; i < n; i++ {
			go func(i int) {
				<-waitChan
				log := domain.CreditLog{
					Key:          fmt.Sprintf("key-19002-%d", i),
					ChangeAmount: 50,
					Biz:          "order",
					BizId:        3,
					Desc:         "购买项目",
				}
				id, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid:  uid,
					Logs: []domain.CreditLog{log},
				})
				resChan <- Result{ID: id, Err: err}
				if err == nil {
					log.ChangeAmount = 0 - log.ChangeAmount
					logChan <- log
				}
			}(i)
		}

		time.Sleep(100 * time.Millisecond)
		close(waitChan)

		expectedLogs := []domain.CreditLog{
			{
				Key:          "key-19002",
				ChangeAmount: 100,
				Biz:          "user",
				BizId:        19002,
				Desc:         "首次注册",
			},
		}
		errCounter := 0
		var expectedID int64
		for i := 0; i < n; i++ {
			res := <-resChan
			if res.Err != nil {
				require.ErrorIs(t, res.Err, service.ErrCreditNotEnough)
				errCounter++
			} else {
				require.NotEqual(t, expectedID, res.ID)
				expectedID = res.ID
			}
		}

		var zeroLog domain.CreditLog
		close(logChan)
		for log := range logChan {
			if log != zeroLog {
				expectedLogs = append(expectedLogs, log)
			}
		}

		require.Equal(t, n-2, errCounter)
		c, err := s.svc.GetCreditsByUID(context.Background(), uid)
		require.NoError(t, err)
		require.Equal(t, uint64(0), c.TotalAmount)
		require.Equal(t, uint64(100), c.LockedTotalAmount)
		require.ElementsMatch(t, c.Logs, expectedLogs)
	})

	t.Run("不同请求_全部成功", func(t *testing.T) {

		n := 10
		changeAmount := 50
		// 创建用户记录
		uid := int64(19003)
		er := s.svc.AddCredits(context.Background(), domain.Credit{
			Uid: uid,
			Logs: []domain.CreditLog{
				{
					Key:          "key-19003",
					ChangeAmount: int64(n * changeAmount),
					Biz:          "user",
					BizId:        19003,
					Desc:         "首次注册",
				},
			},
		})
		require.NoError(t, er)

		waitChan := make(chan struct{})
		resChan := make(chan Result)
		logChan := make(chan domain.CreditLog, n)

		for i := 0; i < n; i++ {
			go func(i int) {
				<-waitChan
				log := domain.CreditLog{
					Key:          fmt.Sprintf("key-19003-%d", i),
					ChangeAmount: int64(changeAmount),
					Biz:          "order",
					BizId:        3,
					Desc:         "购买项目",
				}
				id, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid:  uid,
					Logs: []domain.CreditLog{log},
				})
				resChan <- Result{ID: id, Err: err}
				if err == nil {
					log.ChangeAmount = 0 - log.ChangeAmount
					logChan <- log
				}
			}(i)
		}

		time.Sleep(100 * time.Millisecond)
		close(waitChan)
		expectedLogs := []domain.CreditLog{
			{
				Key:          "key-19003",
				ChangeAmount: int64(n * changeAmount),
				Biz:          "user",
				BizId:        19003,
				Desc:         "首次注册",
			},
		}
		var expectedID int64
		for i := 0; i < n; i++ {
			res := <-resChan
			require.NoError(t, res.Err)
			require.NotEqual(t, expectedID, res.ID)
			expectedID = res.ID
			expectedLogs = append(expectedLogs, <-logChan)
		}
		c, err := s.svc.GetCreditsByUID(context.Background(), uid)
		require.NoError(t, err)
		require.Equal(t, uint64(0), c.TotalAmount)
		require.Equal(t, uint64(n*changeAmount), c.LockedTotalAmount)
		require.ElementsMatch(t, c.Logs, expectedLogs)
	})
}

func (s *ModuleTestSuite) TestService_ConfirmDeductCredits() {
	t := s.T()

	testCases := []struct {
		name           string
		getUIDAndTID   func(t *testing.T) (int64, int64)
		after          func(t *testing.T, uid int64)
		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name: "确认预扣成功_ID有效",
			getUIDAndTID: func(t *testing.T) (int64, int64) {
				t.Helper()
				// 创建已有用户
				uid := int64(8001)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-8001-1",
							ChangeAmount: 100,
							Biz:          "user",
							BizId:        1,
							Desc:         "注册",
						},
					},
				})
				require.NoError(t, err)
				// 预扣
				id, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-8001-2",
							ChangeAmount: 50,
							Biz:          "order",
							BizId:        9,
							Desc:         "购买面试",
						},
					},
				})
				require.NoError(t, err)
				return uid, id
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, uint64(50), c.TotalAmount)
				require.Equal(t, uint64(0), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-8001-2",
						ChangeAmount: -50,
						Biz:          "order",
						BizId:        9,
						Desc:         "购买面试",
					},
					{
						Key:          "key-8001-1",
						ChangeAmount: 100,
						Biz:          "user",
						BizId:        1,
						Desc:         "注册",
					},
				})
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "确认预扣失败_ID有效但非当前用户所有",
			getUIDAndTID: func(t *testing.T) (int64, int64) {
				t.Helper()
				// 创建已有用户
				uid := int64(8002)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-8002-1",
							ChangeAmount: 100,
							Biz:          "user",
							BizId:        1,
							Desc:         "注册",
						},
					},
				})
				require.NoError(t, err)
				return uid, int64(1)
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, uint64(100), c.TotalAmount)
				require.Equal(t, uint64(0), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-8002-1",
						ChangeAmount: 100,
						Biz:          "user",
						BizId:        1,
						Desc:         "注册",
					},
				})
			},
			errRequireFunc: require.Error,
		},
		{
			name: "确认预扣失败_ID为已取消的预扣ID",
			getUIDAndTID: func(t *testing.T) (int64, int64) {
				t.Helper()
				// 创建已有用户
				uid := int64(8003)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-8003-1",
							ChangeAmount: 100,
							Biz:          "user",
							BizId:        1,
							Desc:         "注册",
						},
					},
				})
				require.NoError(t, err)

				// 预扣
				id, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-8003-2",
							ChangeAmount: 50,
							Biz:          "order",
							BizId:        9,
							Desc:         "购买面试",
						},
					},
				})
				require.NoError(t, err)

				// 取消预扣
				require.NoError(t, s.svc.CancelDeductCredits(context.Background(), uid, id))

				return uid, id
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, uint64(100), c.TotalAmount)
				require.Equal(t, uint64(0), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-8003-1",
						ChangeAmount: 100,
						Biz:          "user",
						BizId:        1,
						Desc:         "注册",
					},
				})
			},
			errRequireFunc: require.Error,
		},
		{
			name:           "确认预扣失败_ID非法",
			getUIDAndTID:   func(t *testing.T) (int64, int64) { return int64(8002), int64(1000) },
			after:          func(t *testing.T, uid int64) {},
			errRequireFunc: require.Error,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			uid, tid := tc.getUIDAndTID(t)
			err := s.svc.ConfirmDeductCredits(context.Background(), uid, tid)
			tc.errRequireFunc(t, err)
			tc.after(t, uid)
		})
	}
}

func (s *ModuleTestSuite) TestService_ConfirmDeductCredits_Concurrent() {
	t := s.T()

	// 创建已有用户
	uid := int64(8003)
	err := s.svc.AddCredits(context.Background(), domain.Credit{
		Uid: uid,
		Logs: []domain.CreditLog{
			{
				Key:          "key-8003-1",
				ChangeAmount: 100,
				Biz:          "user",
				BizId:        1,
				Desc:         "注册",
			},
		},
	})
	require.NoError(t, err)
	// 预扣
	tid, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
		Uid: uid,
		Logs: []domain.CreditLog{
			{
				Key:          "key-8003-2",
				ChangeAmount: 20,
				Biz:          "order",
				BizId:        9,
				Desc:         "购买面试",
			},
		},
	})
	require.NoError(t, err)

	n := 10
	waitChan := make(chan struct{})
	errChan := make(chan error)

	for i := 0; i < n; i++ {
		go func(tid int64) {
			<-waitChan
			errChan <- s.svc.ConfirmDeductCredits(context.Background(), uid, tid)
		}(tid)
	}

	time.Sleep(100 * time.Millisecond)
	close(waitChan)
	for i := 0; i < n; i++ {
		require.NoError(t, <-errChan)
	}
	c, err := s.svc.GetCreditsByUID(context.Background(), uid)
	require.NoError(t, err)
	require.Equal(t, uint64(80), c.TotalAmount)
	require.Equal(t, uint64(0), c.LockedTotalAmount)
	require.Equal(t, c.Logs, []domain.CreditLog{
		{
			Key:          "key-8003-2",
			ChangeAmount: -20,
			Biz:          "order",
			BizId:        9,
			Desc:         "购买面试",
		},
		{
			Key:          "key-8003-1",
			ChangeAmount: 100,
			Biz:          "user",
			BizId:        1,
			Desc:         "注册",
		},
	})
}

func (s *ModuleTestSuite) TestService_CancelDeductCredits() {
	t := s.T()

	testCases := []struct {
		name           string
		getUIDAndTID   func(t *testing.T) (int64, int64)
		after          func(t *testing.T, uid int64)
		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name: "取消预扣成功_ID有效且为当前用户所有",
			getUIDAndTID: func(t *testing.T) (int64, int64) {
				t.Helper()
				// 创建已有用户
				uid := int64(9001)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-9001-1",
							ChangeAmount: 100,
							Biz:          "user",
							BizId:        1,
							Desc:         "注册",
						},
					},
				})
				require.NoError(t, err)
				// 预扣
				tid, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-9001-2",
							ChangeAmount: 50,
							Biz:          "order",
							BizId:        9,
							Desc:         "购买面试",
						},
					},
				})
				require.NoError(t, err)
				return uid, tid
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, uint64(100), c.TotalAmount)
				require.Equal(t, uint64(0), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-9001-1",
						ChangeAmount: 100,
						Biz:          "user",
						BizId:        1,
						Desc:         "注册",
					},
				})
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "取消预扣失败_ID为已确认的流水ID",
			getUIDAndTID: func(t *testing.T) (int64, int64) {
				t.Helper()
				// 创建已有用户
				uid := int64(9003)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-9003-1",
							ChangeAmount: 100,
							Biz:          "user",
							BizId:        1,
							Desc:         "注册",
						},
					},
				})
				require.NoError(t, err)
				// 预扣
				tid, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-9003-2",
							ChangeAmount: 50,
							Biz:          "order",
							BizId:        9,
							Desc:         "购买面试",
						},
					},
				})
				require.NoError(t, err)
				// 已确认
				require.NoError(t, s.svc.ConfirmDeductCredits(context.Background(), uid, tid))
				return uid, tid
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, uint64(50), c.TotalAmount)
				require.Equal(t, uint64(0), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-9003-2",
						ChangeAmount: -50,
						Biz:          "order",
						BizId:        9,
						Desc:         "购买面试",
					},
					{
						Key:          "key-9003-1",
						ChangeAmount: 100,
						Biz:          "user",
						BizId:        1,
						Desc:         "注册",
					},
				})
			},
			errRequireFunc: require.Error,
		},
		{
			name: "取消预扣'成功'_ID有效但不为当前用户所有_不返回错误",
			getUIDAndTID: func(t *testing.T) (int64, int64) {
				t.Helper()
				// 创建已有用户
				uid := int64(9002)
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-9002-1",
							ChangeAmount: 100,
							Biz:          "user",
							BizId:        1,
							Desc:         "注册",
						},
					},
				})
				require.NoError(t, err)
				return uid, int64(1)
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()
				c, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, uint64(100), c.TotalAmount)
				require.Equal(t, uint64(0), c.LockedTotalAmount)
				require.Equal(t, c.Logs, []domain.CreditLog{
					{
						Key:          "key-9002-1",
						ChangeAmount: 100,
						Biz:          "user",
						BizId:        1,
						Desc:         "注册",
					},
				})
			},
			errRequireFunc: require.NoError,
		},
		{
			name:           "取消预扣'成功'_ID非法_不返回错误",
			getUIDAndTID:   func(t *testing.T) (int64, int64) { return int64(9004), int64(2000) },
			after:          func(t *testing.T, uid int64) {},
			errRequireFunc: require.NoError,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			uid, tid := tc.getUIDAndTID(t)
			err := s.svc.CancelDeductCredits(context.Background(), uid, tid)
			tc.errRequireFunc(t, err)
			tc.after(t, uid)
		})
	}
}

func (s *ModuleTestSuite) TestService_CancelDeductCredits_Concurrent() {
	t := s.T()
	// 创建已有用户
	uid := int64(9003)
	err := s.svc.AddCredits(context.Background(), domain.Credit{
		Uid: uid,
		Logs: []domain.CreditLog{
			{
				Key:          "key-9003-1",
				ChangeAmount: 100,
				Biz:          "user",
				BizId:        1,
				Desc:         "注册",
			},
		},
	})
	require.NoError(t, err)
	// 预扣
	tid, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
		Uid: uid,
		Logs: []domain.CreditLog{
			{
				Key:          "key-9003-2",
				ChangeAmount: 50,
				Biz:          "order",
				BizId:        9,
				Desc:         "购买面试",
			},
		},
	})
	require.NoError(t, err)

	n := 10
	waitChan := make(chan struct{})
	errChan := make(chan error)

	for i := 0; i < n; i++ {
		go func() {
			<-waitChan
			errChan <- s.svc.CancelDeductCredits(context.Background(), uid, tid)
		}()
	}

	time.Sleep(100 * time.Millisecond)
	close(waitChan)
	for i := 0; i < n; i++ {
		require.NoError(t, <-errChan)
	}
	c, err := s.svc.GetCreditsByUID(context.Background(), uid)
	require.NoError(t, err)
	require.Equal(t, uint64(100), c.TotalAmount)
	require.Equal(t, uint64(0), c.LockedTotalAmount)
	require.Equal(t, c.Logs, []domain.CreditLog{
		{
			Key:          "key-9003-1",
			ChangeAmount: 100,
			Biz:          "user",
			BizId:        1,
			Desc:         "注册",
		},
	})
}

func (s *ModuleTestSuite) TestService_GetCreditsByUID() {
	t := s.T()

	testCases := []struct {
		name string

		before         func(t *testing.T, credit domain.Credit)
		credit         domain.Credit
		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name:   "无记录用户",
			before: func(t *testing.T, credit domain.Credit) {},
			credit: domain.Credit{
				Uid: 20000,
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "有记录用户_无预扣积分",
			before: func(t *testing.T, credit domain.Credit) {
				t.Helper()

				// 创建已有用户
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: credit.Uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-20001-1",
							ChangeAmount: 100,
							Biz:          "Marketing",
							BizId:        2,
							Desc:         "邀请注册",
						},
					},
				})
				require.NoError(t, err)

			},
			credit: domain.Credit{
				Uid:               20001,
				TotalAmount:       100,
				LockedTotalAmount: 0,
				Logs: []domain.CreditLog{
					{
						Key:          "key-20001-1",
						ChangeAmount: 100,
						Biz:          "Marketing",
						BizId:        2,
						Desc:         "邀请注册",
					},
				},
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "有记录用户_有预扣积分",
			before: func(t *testing.T, credit domain.Credit) {
				t.Helper()

				// 创建已有用户
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: credit.Uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-20002-1",
							ChangeAmount: 100,
							Biz:          "Marketing",
							BizId:        2,
							Desc:         "邀请注册",
						},
					},
				})
				require.NoError(t, err)

				// 预扣
				_, err = s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid: credit.Uid,
					Logs: []domain.CreditLog{
						{
							Key:          "key-20002-2",
							ChangeAmount: 50,
							Biz:          "order",
							BizId:        9,
							Desc:         "购买面试",
						},
					},
				})

			},
			credit: domain.Credit{
				Uid:               20002,
				TotalAmount:       50,
				LockedTotalAmount: 50,
				Logs: []domain.CreditLog{
					{
						Key:          "key-20002-2",
						ChangeAmount: -50,
						Biz:          "order",
						BizId:        9,
						Desc:         "购买面试",
					},
					{
						Key:          "key-20002-1",
						ChangeAmount: 100,
						Biz:          "Marketing",
						BizId:        2,
						Desc:         "邀请注册",
					},
				},
			},
			errRequireFunc: require.NoError,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t, tc.credit)
			c, err := s.svc.GetCreditsByUID(context.Background(), tc.credit.Uid)
			tc.errRequireFunc(t, err)
			if err == nil {
				require.Equal(t, tc.credit, c)
			}
		})
	}

}

func (s *ModuleTestSuite) TestHandler_QueryCredits() {
	t := s.T()

	testCases := []struct {
		name string

		before   func(t *testing.T)
		after    func(t *testing.T)
		wantCode int
		wantResp test.Result[web.Credit]
	}{
		{
			name: "用户有记录_有预扣",
			before: func(t *testing.T) {
				t.Helper()

				// 创建已有用户
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: testUID,
					Logs: []domain.CreditLog{
						{
							Key:          "key-10001-1",
							ChangeAmount: 100,
							Biz:          "Marketing",
							BizId:        2,
							Desc:         "邀请注册",
						},
					},
				})
				require.NoError(t, err)

				// 预扣
				id, err := s.svc.TryDeductCredits(context.Background(), domain.Credit{
					Uid: testUID,
					Logs: []domain.CreditLog{
						{
							Key:          "key-10001-2",
							ChangeAmount: 50,
							Biz:          "order",
							BizId:        9,
							Desc:         "购买面试",
						},
					},
				})
				require.NoError(t, err)
				require.NotZero(t, id)

			},
			after: func(t *testing.T) {
				t.Helper()
				s.TearDownTest()
			},
			wantCode: 200,
			wantResp: test.Result[web.Credit]{
				Data: web.Credit{Amount: uint64(50)},
			},
		},
		{
			name: "用户有记录_无预扣",
			before: func(t *testing.T) {
				t.Helper()
				// 创建已有用户
				err := s.svc.AddCredits(context.Background(), domain.Credit{
					Uid: testUID,
					Logs: []domain.CreditLog{
						{
							Key:          "key-10002-1",
							ChangeAmount: 100,
							Biz:          "Marketing",
							BizId:        2,
							Desc:         "邀请注册",
						},
					},
				})
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				t.Helper()
				s.TearDownTest()
			},
			wantCode: 200,
			wantResp: test.Result[web.Credit]{
				Data: web.Credit{Amount: uint64(100)},
			},
		},
		{
			name:     "用户无记录",
			before:   func(t *testing.T) {},
			after:    func(t *testing.T) {},
			wantCode: 200,
			wantResp: test.Result[web.Credit]{
				Data: web.Credit{Amount: uint64(0)},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/credit/detail", nil)
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Credit]()
			s.server.ServeHTTP(recorder, req)
			require.Equal(t, tc.wantCode, recorder.Code)
			require.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
		})
	}
}
