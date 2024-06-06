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
	err = s.db.Exec("DROP TABLE `member_records`").Error
	require.NoError(s.T(), err)
}

func (s *ModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `members`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE TABLE `member_records`").Error
	require.NoError(s.T(), err)
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

	for _, tc := range testCases {
		tc := tc
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

func (s *ModuleTestSuite) newMemberEventMessage(t *testing.T, evt event.MemberEvent) *mq.Message {
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) TestService_ActivateMembership_Concurrent() {
	t := s.T()

	t.Run("相同消息并发测试_只有一个成功", func(t *testing.T) {
		n := 10
		uid := int64(2100)

		waitChan := make(chan struct{})
		errChan := make(chan error)

		for i := 0; i < n; i++ {
			go func() {
				<-waitChan
				err := s.svc.ActivateMembership(context.Background(), domain.Member{
					Uid: uid,
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
				errChan <- err
			}()
		}

		time.Sleep(100 * time.Millisecond)
		close(waitChan)
		errCounter := 0
		for i := 0; i < n; i++ {
			err := <-errChan
			if err != nil {
				require.ErrorIs(t, err, service.ErrDuplicatedMemberRecord)
				errCounter++
			}
		}
		require.Equal(t, n-1, errCounter)

		info, err := s.svc.GetMembershipInfo(context.Background(), uid)
		require.NoError(t, err)

		nowDate := time.Now().UTC()
		startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
		require.True(t, startAt <= info.EndAt, fmt.Sprintf("n = %d, e = %d\n", startAt, info.EndAt))
		require.Equal(t, []domain.MemberRecord{
			{
				Key:   "Key-Same-Message",
				Days:  100,
				Biz:   "相同消息并发测试",
				BizId: 11,
				Desc:  "相同消息并发测试",
			},
		}, info.Records)
		require.Equal(t, uint64(100), uint64(time.Duration(info.EndAt-startAt)*time.Millisecond/(time.Hour*24)))
	})

	t.Run("不同消息并发测试_全部成功", func(t *testing.T) {
		n := 10
		days := uint64(200)
		uid := int64(2200)
		waitChan := make(chan struct{})
		errChan := make(chan error, n)
		recordChan := make(chan domain.MemberRecord, n)
		for i := 0; i < n; i++ {
			go func(i int) {
				<-waitChan
				record := domain.MemberRecord{
					Key:   fmt.Sprintf("Key-diff-Message-%d", i),
					Days:  days,
					Biz:   "不同消息并发测试",
					BizId: 12,
					Desc:  "不同消息并发测试",
				}
				err := s.svc.ActivateMembership(context.Background(), domain.Member{
					Uid:     uid,
					Records: []domain.MemberRecord{record},
				})
				errChan <- err
				recordChan <- record
			}(i)
		}
		time.Sleep(100 * time.Millisecond)
		close(waitChan)
		expectedRecords := make([]domain.MemberRecord, 0, n)
		for i := 0; i < n; i++ {
			require.NoError(t, <-errChan)
			expectedRecords = append(expectedRecords, <-recordChan)
		}
		info, err := s.svc.GetMembershipInfo(context.Background(), uid)
		require.NoError(t, err)
		nowDate := time.Now().UTC()
		startAt := time.Date(nowDate.Year(), nowDate.Month(), nowDate.Day(), 23, 59, 59, 0, time.UTC).UnixMilli()
		require.True(t, startAt <= info.EndAt, fmt.Sprintf("n = %d, e = %d\n", startAt, info.EndAt))
		require.ElementsMatch(t, expectedRecords, info.Records)
		require.Equal(t, uint64(n)*days, uint64(time.Duration(info.EndAt-startAt)*time.Millisecond/(time.Hour*24)))
	})

}

func (s *ModuleTestSuite) TestService_GetMembershipInfo() {
	t := s.T()

	t.Run("不报错_当用户没有会员信息时", func(t *testing.T) {
		uid := int64(99999999)
		info, err := s.svc.GetMembershipInfo(context.Background(), uid)
		require.NoError(t, err)
		require.Equal(t, domain.Member{Records: []domain.MemberRecord{}}, info)
	})
}
