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
	err := s.db.Exec("DROP TABLE `members`").Error
	require.NoError(s.T(), err)
}

func (s *ModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `members`").Error
	require.NoError(s.T(), err)
}

func (s *ModuleTestSuite) TestConsumer_ConsumeRegistrationEvent() {
	t := s.T()
	producer, er := s.mq.Producer("user_registration_events")
	require.NoError(t, er)

	testCases := []struct {
		name   string
		before func(t *testing.T, producer mq.Producer, message *mq.Message)
		after  func(t *testing.T, uid int64)

		evt            event.RegistrationEvent
		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name: "开会员成功_新用户注册",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)
				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()

				info, err := s.svc.GetMembershipInfo(context.Background(), uid)
				require.NoError(t, err)

				nowDate := time.Now().UTC()
				startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
				endAt := time.Date(2024, 6, 30, 23, 59, 59, 0, time.UTC).UnixMilli()

				require.Equal(t, endAt, info.EndAt)

				require.True(t, startAt <= info.EndAt, fmt.Sprintf("n = %d, e = %d\n", startAt, info.EndAt))
				for i := range info.Records {
					require.NotZero(t, info.Records[i].Key)
					info.Records[i].Key = ""
				}
				require.Equal(t, []domain.MemberRecord{
					{
						Biz:   "user",
						BizId: uid,
						Desc:  "注册福利",
						Days:  uint64(time.Duration(info.EndAt-startAt) * time.Millisecond / (time.Hour * 24)),
					},
				}, info.Records)
			},
			evt: event.RegistrationEvent{
				Uid: 1991,
			},
			errRequireFunc: require.NoError,
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
							Biz:   "user",
							BizId: 1993,
							Desc:  "注册福利",
							Days:  uint64(time.Duration(endAt-startAt) * time.Millisecond / (time.Hour * 24)),
						},
					},
				})
				require.NoError(t, err)

				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)

				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, uid int64) {},
			evt: event.RegistrationEvent{
				Uid: 1991,
			},
			errRequireFunc: require.Error,
		},
	}

	consumer, err := event.NewRegistrationEventConsumer(s.svc, s.mq)
	require.NoError(t, err)

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			message := s.newRegistrationEventMessage(t, tc.evt)
			tc.before(t, producer, message)

			err = consumer.Consume(context.Background())
			tc.errRequireFunc(t, err)

			err = consumer.Consume(context.Background())
			require.Error(t, err)

			tc.after(t, tc.evt.Uid)
		})
	}
}

func (s *ModuleTestSuite) TestConsumer_ConsumeMemberEvent() {
	t := s.T()

	producer, er := s.mq.Producer("member_update_events")
	require.NoError(t, er)

	testCases := []struct {
		name   string
		before func(t *testing.T, producer mq.Producer, message *mq.Message)
		after  func(t *testing.T, uid int64)

		evt            event.MemberEvent
		errRequireFunc require.ErrorAssertionFunc
	}{
		{
			name: "开会员成功_新注册用户_创建记录",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)

				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()

				info, err := s.svc.GetMembershipInfo(context.Background(), uid)
				require.NoError(t, err)

				nowDate := time.Now().UTC()
				startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
				require.True(t, startAt <= info.EndAt, fmt.Sprintf("n = %d, e = %d\n", startAt, info.EndAt))
				require.Equal(t, []domain.MemberRecord{
					{
						Key:   "member-key-20001",
						Days:  uint64(time.Duration(info.EndAt-startAt) * time.Millisecond / (time.Hour * 24)),
						Biz:   "user",
						BizId: uid,
						Desc:  "新注册用户",
					},
				}, info.Records)
			},
			evt: event.MemberEvent{
				Key:    "member-key-20001",
				Uid:    20001,
				Days:   30,
				Biz:    "user",
				BizId:  20001,
				Action: "新注册用户",
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "开会员成功_会员已过期_重新开启",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()

				uid := int64(20002)

				// 创建过期会员相关记录
				now := time.Date(2024, 1, 1, 18, 18, 1, 0, time.UTC).UnixMilli()
				require.NoError(t, s.db.Create(&dao.Member{
					Uid:     uid,
					EndAt:   time.Date(2024, 2, 1, 23, 59, 59, 0, time.UTC).UnixMilli(),
					Version: 1,
					Ctime:   now,
					Utime:   now,
				}).Error)

				require.NoError(t, s.db.Create(&dao.MemberRecord{
					Key:   "member-key-20002-1",
					Uid:   uid,
					Biz:   "user",
					BizId: uid,
					Desc:  "首次注册",
					Days:  31,
					Ctime: now,
					Utime: now,
				}).Error)

				_, err := producer.Produce(context.Background(), message)
				require.NoError(t, err)

				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()

				info, err := s.svc.GetMembershipInfo(context.Background(), uid)
				require.NoError(t, err)

				nowDate := time.Now().UTC()
				startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
				require.True(t, startAt <= info.EndAt, fmt.Sprintf("n = %d, e = %d\n", startAt, info.EndAt))
				require.Equal(t, []domain.MemberRecord{
					{
						Key:   "member-key-20002-2",
						Days:  31,
						Biz:   "order",
						BizId: 2,
						Desc:  "购买月会员",
					},
					{
						Key:   "member-key-20002-1",
						Days:  31,
						Biz:   "user",
						BizId: uid,
						Desc:  "首次注册",
					},
				}, info.Records)
				require.Equal(t, uint64(31), uint64(time.Duration(info.EndAt-startAt)*time.Millisecond/(time.Hour*24)))

			},
			evt: event.MemberEvent{
				Key:    "member-key-20002-2",
				Uid:    20002,
				Days:   31,
				Biz:    "order",
				BizId:  2,
				Action: "购买月会员",
			},
			errRequireFunc: require.NoError,
		},
		{
			name: "开会员成功_会员生效中_续约成功",
			before: func(t *testing.T, producer mq.Producer, message *mq.Message) {
				t.Helper()

				uid := int64(20003)
				err := s.svc.ActivateMembership(context.Background(), domain.Member{
					Uid: uid,
					Records: []domain.MemberRecord{
						{
							Key:   "member-key-20003-1",
							Biz:   "user",
							BizId: uid,
							Desc:  "首次注册",
							Days:  31,
						},
					},
				})
				require.NoError(t, err)

				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)

				// 模拟重试
				_, err = producer.Produce(context.Background(), message)
				require.NoError(t, err)
			},
			after: func(t *testing.T, uid int64) {
				t.Helper()

				info, err := s.svc.GetMembershipInfo(context.Background(), uid)
				require.NoError(t, err)

				nowDate := time.Now().UTC()
				startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
				require.True(t, startAt <= info.EndAt, fmt.Sprintf("n = %d, e = %d\n", startAt, info.EndAt))
				require.Equal(t, []domain.MemberRecord{
					{
						Key:   "member-key-20003-2",
						Days:  365,
						Biz:   "order",
						BizId: 2,
						Desc:  "购买年会员",
					},
					{
						Key:   "member-key-20003-1",
						Days:  31,
						Biz:   "user",
						BizId: uid,
						Desc:  "首次注册",
					},
				}, info.Records)
				require.Equal(t, uint64(31+365), uint64(time.Duration(info.EndAt-startAt)*time.Millisecond/(time.Hour*24)))

			},
			evt: event.MemberEvent{
				Key:    "member-key-20003-2",
				Uid:    20003,
				Days:   365,
				Biz:    "order",
				BizId:  2,
				Action: "购买年会员",
			},
			errRequireFunc: require.NoError,
		},
	}

	consumer, err := event.NewMemberEventConsumer(s.svc, s.mq)
	require.NoError(t, err)

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.name, func(t *testing.T) {
			message := s.newMemberEventMessage(t, tc.evt)
			tc.before(t, producer, message)

			err = consumer.Consume(context.Background())
			tc.errRequireFunc(t, err)

			// 处理重复消息
			err = consumer.Consume(context.Background())
			require.NoError(t, err)

			tc.after(t, tc.evt.Uid)
		})
	}
}

func (s *ModuleTestSuite) newRegistrationEventMessage(t *testing.T, evt event.RegistrationEvent) *mq.Message {
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) newMemberEventMessage(t *testing.T, evt event.MemberEvent) *mq.Message {
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) TestService_ActivateMembership() {
	t := s.T()

	t.Run("相同消息并发测试", func(t *testing.T) {
		n := 10

		waitChan := make(chan struct{})
		errChan := make(chan error)

		for i := 0; i < n; i++ {
			i := i
			go func() {
				<-waitChan
				err := s.svc.ActivateMembership(context.Background(), domain.Member{
					Uid: 2100,
					Records: []domain.MemberRecord{
						{
							Key:   "Key-Same-Message",
							Days:  100,
							Biz:   "相同消息并发测试",
							BizId: 11,
							Desc:  "相同消息并发测试",
						},
					},
				})
				t.Logf("invoked i = %d\n", i)
				errChan <- err
			}()
		}

		time.Sleep(100 * time.Millisecond)
		close(waitChan)
		counter := 0
		for i := 0; i < n; i++ {
			err := <-errChan
			if err != nil {
				require.ErrorIs(t, err, service.ErrDuplicatedMemberRecord)
				counter++
			}
		}
		require.Equal(t, n-1, counter)
	})

	t.Run("不同消息并发测试", func(t *testing.T) {
		n := 10
		waitChan := make(chan struct{})
		errChan := make(chan error)
		for i := 0; i < n; i++ {
			go func(i int) {
				<-waitChan
				err := s.svc.ActivateMembership(context.Background(), domain.Member{
					Uid: 2200,
					Records: []domain.MemberRecord{
						{
							Key:   fmt.Sprintf("Key-diff-Message-%d", i),
							Days:  200,
							Biz:   "不同消息并发测试",
							BizId: 12,
							Desc:  "不同消息并发测试",
						},
					},
				})
				t.Logf("invoked j = %d\n", i)
				errChan <- err
			}(i)
		}
		time.Sleep(100 * time.Millisecond)
		close(waitChan)
		for i := 0; i < n; i++ {
			require.NoError(t, <-errChan)
		}
	})

}
