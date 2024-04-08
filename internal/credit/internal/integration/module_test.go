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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ModuleTestSuite struct {
	suite.Suite
	db  *egorm.Component
	mq  mq.MQ
	svc service.Service
}

func TestModule(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
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

func (s *ModuleTestSuite) TestConsumer_ConsumeCreditEvent() {
	t := s.T()
	producer, err := s.mq.Producer("credit_events")
	require.NoError(t, err)

	testCases := []struct {
		name   string
		before func(t *testing.T, producer mq.Producer, message *mq.Message)
		after  func(t *testing.T, uid int64)

		Uid           int64
		errAssertFunc assert.ErrorAssertionFunc
	}{
		{
			name: "开会员成功_新用户注册",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, uid int64) {

			},
			Uid:           1991,
			errAssertFunc: assert.NoError,
		},
		{
			name: "开会员失败_用户已注册_会员生效中",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)

			},
			after:         func(t *testing.T, uid int64) {},
			Uid:           1993,
			errAssertFunc: assert.Error,
		},
		{
			name: "开会员失败_用户已注册_会员已失效",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after:         func(t *testing.T, uid int64) {},
			Uid:           1994,
			errAssertFunc: assert.Error,
		},
	}

	consumer, err := event.NewCreditConsumer(s.svc, s.mq)
	require.NoError(t, err)

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			message := s.newRegistrationEventMessage(t, tc.Uid)
			tc.before(t, producer, message)

			err = consumer.Consume(context.Background())

			tc.errAssertFunc(t, err)
			tc.after(t, tc.Uid)
		})
	}
}

func (s *ModuleTestSuite) newRegistrationEventMessage(t *testing.T, uid int64) *mq.Message {
	marshal, err := json.Marshal(event.CreditEvent{Uid: uid})
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) TestService_AddCreditsAndGetCreditsByUID() {
	t := s.T()

	testCases := []struct {
		name string

		before func(t *testing.T)
		uid    int64
		amount int64
		after  func(t *testing.T)

		wantErr error
	}{
		{
			name: "用户积分主记录不存在_增加积分成功",
			before: func(t *testing.T) {
				t.Helper()
				_, err := s.svc.GetCreditsByUID(context.Background(), 5127)
				require.Error(t, err)
			},
			uid:    5127,
			amount: 51,
			after: func(t *testing.T) {
				t.Helper()
				credits, err := s.svc.GetCreditsByUID(context.Background(), 5127)
				require.NoError(t, err)
				require.Equal(t, int64(51), credits)
			},
		},
		{
			name: "用户积分主记录已存在_无预扣积分_增加积分成功",
			before: func(t *testing.T) {
				t.Helper()
				uid := int64(5128)

				_, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.Error(t, err)

				amount := int64(199)
				require.NoError(t, s.svc.AddCredits(context.Background(), domain.Credit{
					Uid:    uid,
					Amount: amount,
				}))

				credits, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, amount, credits)
			},
			uid:    5128,
			amount: 100,
			after: func(t *testing.T) {
				t.Helper()
				uid := int64(5128)
				credits, err := s.svc.GetCreditsByUID(context.Background(), uid)
				require.NoError(t, err)
				require.Equal(t, int64(299), credits)
			},
		},
		{
			name: "用户积分主记录已存在_有预扣积分_增加积分成功",
		},
		// todo: 用户积分主记录已存在_有预扣积分_增加积分成功
		// todo: amount <= 0

	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			tt.before(t)
			err := s.svc.AddCredits(context.Background(), domain.Credit{
				Uid:    tt.uid,
				Amount: tt.amount,
			})
			require.Equal(t, tt.wantErr, err)
			tt.after(t)
		})
	}
}
