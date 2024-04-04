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
	"time"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member/internal/domain"
	"github.com/ecodeclub/webook/internal/member/internal/event"
	"github.com/ecodeclub/webook/internal/member/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/member/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/member/internal/service"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestMemberIntegrationTest(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

type ModuleTestSuite struct {
	suite.Suite
	db  *egorm.Component
	mq  mq.MQ
	svc service.Service
	dao dao.MemberDAO
}

func (s *ModuleTestSuite) SetupSuite() {
	s.svc = startup.InitService()
	s.db = testioc.InitDB()
	require.NoError(s.T(), dao.InitTables(s.db))
	s.dao = dao.NewMemberGORMDAO(s.db)

	s.mq = testioc.InitMQ()
}

func (s *ModuleTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `members`").Error
	require.NoError(s.T(), err)
}

func (s *ModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `members`").Error
	require.NoError(s.T(), err)
}

func (s *ModuleTestSuite) TestConsumer_ConsumeRegistrationEvent() {
	t := s.T()

	topic := "test_user_registration_events"
	err := s.mq.CreateTopic(context.Background(), topic, 1)

	t.Cleanup(func() {
		_ = s.mq.DeleteTopics(context.Background(), topic)
	})
	require.NoError(t, err)

	producer, err := s.mq.Producer(topic)
	require.NoError(t, err)

	consumer, err := s.mq.Consumer(topic, topic)
	require.NoError(t, err)

	testCases := map[string]struct {
		before func(t *testing.T, producer mq.Producer, message *mq.Message)
		after  func(t *testing.T, uid int64, startAtFunc func() int64, endAtFunc func() int64)

		UserID        int64
		startAtFunc   func() int64
		endAtFunc     func() int64
		errAssertFunc assert.ErrorAssertionFunc
	}{
		"开会员成功_新用户注册": {
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, uid int64, startAtFunc func() int64, endAtFunc func() int64) {
				t.Helper()

				info, err := s.svc.GetMembershipInfo(context.Background(), uid)
				require.NoError(t, err)
				require.NotZero(t, info.ID)
				require.Equal(t, int64(domain.MemberStatusActive), info.Status)
				require.Equal(t, startAtFunc(), info.StartAt)
				require.Equal(t, endAtFunc(), info.EndAt)
			},
			UserID: 1991,
			startAtFunc: func() int64 {
				return time.Date(2024, 5, 10, 18, 24, 33, 0, time.Local).Unix()
			},
			endAtFunc: func() int64 {
				return time.Date(2024, 6, 30, 23, 59, 59, 0, time.Local).Unix()
			},
			errAssertFunc: assert.NoError,
		},
		"开会员成功_新用户注册_优惠已到期": {
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after:  func(t *testing.T, uid int64, startAtFunc func() int64, endAtFunc func() int64) {},
			UserID: 1992,
			startAtFunc: func() int64 {
				return time.Date(2024, 7, 1, 18, 24, 33, 0, time.Local).Unix()
			},
			endAtFunc: func() int64 {
				return time.Date(2024, 6, 30, 23, 59, 59, 0, time.Local).Unix()
			},
			errAssertFunc: assert.Error,
		},
		"开会员失败_用户已注册_会员生效中": {
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)

				_, err = s.svc.CreateNewMembership(context.Background(), domain.Member{
					UserID:  1993,
					StartAt: time.Now().Local().Unix(),
					EndAt:   time.Now().Add(time.Hour * 24 * 30).Local().Unix(),
					Status:  domain.MemberStatusActive,
				})
				require.NoError(t, err)
			},
			after:         func(t *testing.T, uid int64, startAtFunc func() int64, endAtFunc func() int64) {},
			UserID:        1993,
			startAtFunc:   nil,
			endAtFunc:     nil,
			errAssertFunc: assert.Error,
		},
		"开会员失败_用户已注册_会员已失效": {
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)

				_, err = s.svc.CreateNewMembership(context.Background(), domain.Member{
					UserID:  1994,
					StartAt: time.Date(2023, 4, 11, 18, 24, 33, 0, time.Local).Unix(),
					EndAt:   time.Date(2023, 6, 30, 23, 59, 59, 0, time.Local).Unix(),
					Status:  domain.MemberStatusDeactivate,
				})
				require.NoError(t, err)

			},
			after:         func(t *testing.T, uid int64, startAtFunc func() int64, endAtFunc func() int64) {},
			UserID:        1994,
			startAtFunc:   nil,
			endAtFunc:     nil,
			errAssertFunc: assert.Error,
		},
	}

	for i := range testCases {
		name, tc := i, testCases[i]

		t.Run(name, func(t *testing.T) {

			message := s.newRegistrationEventMessage(t, tc.UserID)
			tc.before(t, producer, message)

			evtConsumer := event.NewMQConsumer(s.svc, consumer, tc.startAtFunc, tc.endAtFunc)
			err = evtConsumer.ConsumeRegistrationEvent(context.Background())
			tc.errAssertFunc(t, err)

			tc.after(t, tc.UserID, tc.startAtFunc, tc.endAtFunc)
		})
	}
}

func (s *ModuleTestSuite) newRegistrationEventMessage(t *testing.T, userID int64) *mq.Message {
	marshal, err := json.Marshal(event.RegistrationEvent{UserID: userID})
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}
