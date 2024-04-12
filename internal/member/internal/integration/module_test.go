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

func TestMemberModule(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

type ModuleTestSuite struct {
	suite.Suite
	db  *egorm.Component
	mq  mq.MQ
	svc service.Service
}

func (s *ModuleTestSuite) SetupSuite() {
	s.svc = startup.InitService()
	s.db = testioc.InitDB()
	s.mq = testioc.InitMQ()
}

func (s *ModuleTestSuite) TearDownSuite() {
	// err := s.db.Exec("DROP TABLE `members`").Error
	// require.NoError(s.T(), err)
}

func (s *ModuleTestSuite) TearDownTest() {
	// err := s.db.Exec("TRUNCATE TABLE `members`").Error
	// require.NoError(s.T(), err)
}

func (s *ModuleTestSuite) TestConsumer_ConsumeRegistrationEvent() {
	t := s.T()
	producer, er := s.mq.Producer("user_registration_events")
	require.NoError(t, er)

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
				t.Helper()

				info, err := s.svc.GetMembershipInfo(context.Background(), uid)
				require.NoError(t, err)

				nowDate := time.Now().UTC()
				startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
				endAt := time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC).UnixMilli()

				require.Equal(t, startAt, info.StartAt)
				require.Equal(t, endAt, info.EndAt)

				require.True(t, startAt <= info.EndAt, fmt.Sprintf("n = %d, e = %d\n", startAt, info.EndAt))
				for i := range info.Records {
					require.NotZero(t, info.Records[i].Key)
					info.Records[i].Key = ""
				}
				require.Equal(t, []domain.MemberRecord{
					{
						Biz:   1,
						BizId: uid,
						Desc:  "注册福利",
						Days:  uint64(time.Duration(info.EndAt-startAt) * time.Millisecond / (time.Hour * 24)),
					},
				}, info.Records)
			},
			Uid:           1991,
			errAssertFunc: assert.NoError,
		},

		// 开会员失败_新用户注册_优惠已到期, 无法测到,通过代码审查来补充
		{
			name: "开会员失败_用户已注册",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()

				nowDate := time.Now().UTC()
				startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
				endAt := time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC).UnixMilli()

				err := s.svc.ActivateMembership(context.Background(), domain.Member{
					Uid: 1993,
					Records: []domain.MemberRecord{
						{
							Key:   "member-key-1993",
							Biz:   1,
							BizId: 1993,
							Desc:  "注册福利",
							Days:  uint64(time.Duration(endAt-startAt) * time.Millisecond / (time.Hour * 24)),
						},
					},
				})
				require.NoError(t, err)

				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after:         func(t *testing.T, uid int64) {},
			Uid:           1993,
			errAssertFunc: assert.Error,
		},
	}

	consumer, err := event.NewRegistrationEventConsumer(s.svc, s.mq)
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

func (s *ModuleTestSuite) TestService_ActivateMembership() {
	t := s.T()

	var testCases = []struct {
		name string

		before        func(t *testing.T, uid int64)
		member        domain.Member
		errAssertFunc require.ErrorAssertionFunc
		after         func(t *testing.T, uid int64, records []domain.MemberRecord)
	}{
		{
			name:   "开会员成功_新注册用户_创建记录",
			before: func(t *testing.T, uid int64) {},
			member: domain.Member{
				Uid: 20001,
				Records: []domain.MemberRecord{
					{
						Key:   "member-key-20001",
						Biz:   1,
						BizId: 1,
						Desc:  "新注册用户",
						Days:  30,
					},
				},
			},
			errAssertFunc: require.NoError,
			after: func(t *testing.T, uid int64, records []domain.MemberRecord) {
				t.Helper()

				info, err := s.svc.GetMembershipInfo(context.Background(), uid)
				require.NoError(t, err)

				nowDate := time.Now().UTC()
				startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
				require.Equal(t, startAt, info.StartAt)
				require.True(t, startAt < info.EndAt, fmt.Sprintf("n = %d, e = %d\n", startAt, info.EndAt))
				require.Equal(t, records, info.Records)
				require.Equal(t, info.Records[0].Days, uint64(time.Duration(info.EndAt-startAt)*time.Millisecond/(time.Hour*24)))
			},
		},
		{
			name: "开会员成功_会员已过期_重新开启",
			before: func(t *testing.T, uid int64) {
				t.Helper()

				now := time.Date(2024, 1, 1, 18, 18, 1, 0, time.UTC).UnixMilli()
				require.NoError(t, s.db.Create(&dao.Member{
					Uid:     uid,
					StartAt: time.Date(2024, 1, 1, 23, 59, 59, 0, time.UTC).UnixMilli(),
					EndAt:   time.Date(2024, 2, 1, 23, 59, 59, 0, time.UTC).UnixMilli(),
					Version: 1,
					Ctime:   now,
					Utime:   now,
				}).Error)

				require.NoError(t, s.db.Create(&dao.MemberRecord{
					Key:   "member-key-20002-1",
					Uid:   uid,
					Biz:   1,
					BizId: 1,
					Desc:  "首次注册",
					Days:  31,
					Ctime: now,
					Utime: now,
				}).Error)

			},
			member: domain.Member{
				Uid: 20002,
				Records: []domain.MemberRecord{
					{
						Key:   "member-key-20002-2",
						Biz:   2,
						BizId: 2,
						Desc:  "购买月会员",
						Days:  31,
					},
				},
			},
			errAssertFunc: require.NoError,
			after: func(t *testing.T, uid int64, records []domain.MemberRecord) {
				t.Helper()

				info, err := s.svc.GetMembershipInfo(context.Background(), uid)
				require.NoError(t, err)

				nowDate := time.Now().UTC()
				startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
				require.Equal(t, startAt, info.StartAt)
				require.True(t, startAt < info.EndAt, fmt.Sprintf("n = %d, e = %d\n", startAt, info.EndAt))
				require.Equal(t, []domain.MemberRecord{
					records[0],
					{
						Key:   "member-key-20002-1",
						Biz:   1,
						BizId: 1,
						Desc:  "首次注册",
						Days:  31,
					},
				}, info.Records)
				require.Equal(t, info.Records[0].Days, uint64(time.Duration(info.EndAt-startAt)*time.Millisecond/(time.Hour*24)))
			},
		},
		{
			name: "开会员成功_会员生效中_续约成功",
			before: func(t *testing.T, uid int64) {
				t.Helper()

				err := s.svc.ActivateMembership(context.Background(), domain.Member{
					Uid: uid,
					Records: []domain.MemberRecord{
						{
							Key:   "member-key-20003-1",
							Biz:   1,
							BizId: 1,
							Desc:  "首次注册",
							Days:  31,
						},
					},
				})
				require.NoError(t, err)
			},
			member: domain.Member{
				Uid: 20003,
				Records: []domain.MemberRecord{
					{
						Key:   "member-key-20003-2",
						Biz:   2,
						BizId: 2,
						Desc:  "购买年会员",
						Days:  365,
					},
				},
			},
			errAssertFunc: require.NoError,
			after: func(t *testing.T, uid int64, records []domain.MemberRecord) {
				t.Helper()

				info, err := s.svc.GetMembershipInfo(context.Background(), uid)
				require.NoError(t, err)

				nowDate := time.Now().UTC()
				startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
				require.Equal(t, startAt, info.StartAt)
				require.True(t, startAt < info.EndAt, fmt.Sprintf("n = %d, e = %d\n", startAt, info.EndAt))
				require.Equal(t, []domain.MemberRecord{
					records[0],
					{
						Key:   "member-key-20003-1",
						Biz:   1,
						BizId: 1,
						Desc:  "首次注册",
						Days:  31,
					},
				}, info.Records)
				require.Equal(t, info.Records[0].Days+info.Records[1].Days, uint64(time.Duration(info.EndAt-startAt)*time.Millisecond/(time.Hour*24)))
			},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t, tc.member.Uid)
			err := s.svc.ActivateMembership(context.Background(), tc.member)
			tc.errAssertFunc(t, err)
			tc.after(t, tc.member.Uid, tc.member.Records)
		})
	}

}

func (s *ModuleTestSuite) newRegistrationEventMessage(t *testing.T, uid int64) *mq.Message {
	marshal, err := json.Marshal(event.RegistrationEvent{Uid: uid})
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}
