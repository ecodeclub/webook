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
	producer, err := s.mq.Producer("user_registration_events")
	require.NoError(t, err)

	testCases := map[string]struct {
		before func(t *testing.T, producer mq.Producer, message *mq.Message)
		after  func(t *testing.T, uid int64)

		Uid           int64
		errAssertFunc assert.ErrorAssertionFunc
	}{
		"开会员成功_新用户注册": {
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				// ctx, cancel := context.WithTimeout(context.Background(), time.Second * 3)
				// defer cancel()
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, uid int64) {
				// ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				// defer cancel()
				info, err := s.svc.GetMembershipInfo(context.Background(), uid)
				require.NoError(t, err)
				require.NotZero(t, info.ID)
			},
			Uid:           1991,
			errAssertFunc: assert.NoError,
		},
		// 开会员失败_新用户注册_优惠已到期
		"开会员失败_用户已注册_会员生效中": {
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				_, err = s.svc.CreateNewMembership(context.Background(), domain.Member{
					UID:     1993,
					StartAt: time.Now().UnixMilli(),
					EndAt:   time.Now().Add(time.Hour).UnixMilli(),
				})
				require.NoError(t, err)
			},
			after:         func(t *testing.T, uid int64) {},
			Uid:           1993,
			errAssertFunc: assert.Error,
		},
		"开会员失败_用户已注册_会员已失效": {
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				_, err = s.svc.CreateNewMembership(context.Background(), domain.Member{
					UID:     1994,
					StartAt: time.Date(2023, 4, 11, 18, 24, 33, 0, time.UTC).UnixMilli(),
					EndAt:   time.Date(2023, 6, 30, 23, 59, 59, 0, time.UTC).UnixMilli(),
				})
				require.NoError(t, err)
			},
			after:         func(t *testing.T, uid int64) {},
			Uid:           1994,
			errAssertFunc: assert.Error,
		},
	}

	c, err := event.NewRegistrationEventConsumer(s.svc, s.mq)
	require.NoError(t, err)

	for i := range testCases {
		name, tc := i, testCases[i]
		t.Run(name, func(t *testing.T) {
			message := s.newRegistrationEventMessage(t, tc.Uid)
			tc.before(t, producer, message)

			err = c.Consume(context.Background())

			tc.errAssertFunc(t, err)
			tc.after(t, tc.Uid)
		})
	}
}

func (s *ModuleTestSuite) newRegistrationEventMessage(t *testing.T, uid int64) *mq.Message {
	marshal, err := json.Marshal(event.RegistrationEvent{Uid: uid})
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}
