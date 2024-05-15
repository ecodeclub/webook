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
	"github.com/ecodeclub/webook/internal/permission/internal/domain"
	"github.com/ecodeclub/webook/internal/permission/internal/event"
	"github.com/ecodeclub/webook/internal/permission/internal/repository"
	"github.com/ecodeclub/webook/internal/permission/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/permission/internal/service"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/test/mocks"
	"github.com/ego-component/egorm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const testID = 718321

func TestPermissionModule(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

type ModuleTestSuite struct {
	suite.Suite
	db   *egorm.Component
	mq   mq.MQ
	repo repository.PermissionRepository
}

func (s *ModuleTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	s.mq = testioc.InitMQ()
	s.NoError(dao.InitTables(s.db))
	s.repo = repository.NewPermissionRepository(dao.NewPermissionGORMDAO(s.db))

}

func (s *ModuleTestSuite) TearDownSuite() {
	err := s.db.Exec("DROP TABLE `personal_permissions`").Error
	s.NoError(err)
}

func (s *ModuleTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `personal_permissions`").Error
	s.NoError(err)
}

func (s *ModuleTestSuite) TestConsumer_ConsumePermissionEvent() {
	t := s.T()

	testCases := []struct {
		name            string
		before          func(t *testing.T)
		newConsumerFunc func(t *testing.T, ctrl *gomock.Controller, evt event.PermissionEvent) *event.PermissionEventConsumer
		evt             event.PermissionEvent
		after           func(t *testing.T, evt event.PermissionEvent)
		errRequireFunc  require.ErrorAssertionFunc
	}{
		{
			name:   "消费权限消息成功_开通多个权限",
			before: func(tt *testing.T) {},
			newConsumerFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.PermissionEvent) *event.PermissionEventConsumer {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)

				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newPermissionEventMessage(t, evt), nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)

				c, err := event.NewPermissionEventConsumer(service.NewPermissionService(s.repo), mockMQ)
				require.NoError(t, err)
				return c
			},
			evt: event.PermissionEvent{
				UID:    testID,
				Biz:    "project",
				BizIDs: []int64{1, 2, 2, 1},
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, evt event.PermissionEvent) {
				t.Helper()
				permissions, err := s.repo.FindPersonalPermissions(context.Background(), testID)
				require.NoError(t, err)

				require.ElementsMatch(t, []domain.PersonalPermission{
					{
						UID:   testID,
						Biz:   "project",
						BizID: 1,
					},
					{
						UID:   testID,
						Biz:   "project",
						BizID: 2,
					},
				}, permissions)
			},
		},
		{
			name: "消费权限消息成功_全部重复开通多个权限",
			before: func(t *testing.T) {
				t.Helper()
				uid := int64(22971)
				err := s.repo.CreatePersonalPermission(context.Background(), []domain.PersonalPermission{
					{
						UID:   uid,
						Biz:   "interview",
						BizID: 10,
					},
					{
						UID:   uid,
						Biz:   "interview",
						BizID: 12,
					},
				})
				require.NoError(t, err)
			},
			newConsumerFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.PermissionEvent) *event.PermissionEventConsumer {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)

				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newPermissionEventMessage(t, evt), nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)

				c, err := event.NewPermissionEventConsumer(service.NewPermissionService(s.repo), mockMQ)
				require.NoError(t, err)
				return c
			},
			evt: event.PermissionEvent{
				UID:    22971,
				Biz:    "interview",
				BizIDs: []int64{12, 10, 10, 10, 10},
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, evt event.PermissionEvent) {
				t.Helper()
				uid := int64(22971)
				permissions, err := s.repo.FindPersonalPermissions(context.Background(), uid)
				require.NoError(t, err)
				require.ElementsMatch(t, []domain.PersonalPermission{
					{
						UID:   uid,
						Biz:   "interview",
						BizID: 12,
					},
					{
						UID:   uid,
						Biz:   "interview",
						BizID: 10,
					},
				}, permissions)
			},
		},
		{
			name: "消费权限消息成功_部分重复开通多个权限",
			before: func(t *testing.T) {
				t.Helper()
				uid := int64(33977)
				err := s.repo.CreatePersonalPermission(context.Background(), []domain.PersonalPermission{
					{
						UID:   uid,
						Biz:   "interview",
						BizID: 25,
					},
				})
				require.NoError(t, err)
			},
			newConsumerFunc: func(t *testing.T, ctrl *gomock.Controller, evt event.PermissionEvent) *event.PermissionEventConsumer {
				t.Helper()

				mockMQ := mocks.NewMockMQ(ctrl)

				mockConsumer := mocks.NewMockConsumer(ctrl)
				mockConsumer.EXPECT().Consume(gomock.Any()).Return(s.newPermissionEventMessage(t, evt), nil).Times(2)

				mockMQ.EXPECT().Consumer(gomock.Any(), gomock.Any()).Return(mockConsumer, nil)

				c, err := event.NewPermissionEventConsumer(service.NewPermissionService(s.repo), mockMQ)
				require.NoError(t, err)
				return c
			},
			evt: event.PermissionEvent{
				UID:    33977,
				Biz:    "interview",
				BizIDs: []int64{29, 25},
			},
			errRequireFunc: require.NoError,
			after: func(t *testing.T, evt event.PermissionEvent) {
				t.Helper()
				uid := int64(33977)
				permissions, err := s.repo.FindPersonalPermissions(context.Background(), uid)
				require.NoError(t, err)
				require.ElementsMatch(t, []domain.PersonalPermission{
					{
						UID:   uid,
						Biz:   "interview",
						BizID: 29,
					},
					{
						UID:   uid,
						Biz:   "interview",
						BizID: 25,
					},
				}, permissions)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			consumer := tc.newConsumerFunc(t, ctrl, tc.evt)

			err := consumer.Consume(context.Background())
			tc.errRequireFunc(t, err)

			err = consumer.Consume(context.Background())
			tc.errRequireFunc(t, err)

			tc.after(t, tc.evt)
		})
	}
}

func (s *ModuleTestSuite) newPermissionEventMessage(t *testing.T, evt event.PermissionEvent) *mq.Message {
	marshal, err := json.Marshal(evt)
	require.NoError(t, err)
	return &mq.Message{Value: marshal}
}

func (s *ModuleTestSuite) TestService_HasPersonalPermission() {
	t := s.T()

	testCases := []struct {
		name       string
		before     func(t *testing.T)
		newSvcFunc func(t *testing.T) service.Service
		req        domain.PersonalPermission

		wantResult bool
		wantErr    error
	}{
		{
			name: "检查用户权限_有权限",
			before: func(t *testing.T) {
				t.Helper()
				var err = s.repo.CreatePersonalPermission(context.Background(), []domain.PersonalPermission{
					{
						UID:   testID,
						Biz:   "ai",
						BizID: 47,
					},
				})
				require.NoError(t, err)
			},
			newSvcFunc: func(t *testing.T) service.Service {
				t.Helper()
				return service.NewPermissionService(s.repo)
			},
			req: domain.PersonalPermission{
				UID:   testID,
				Biz:   "ai",
				BizID: 47,
			},
			wantResult: true,
			wantErr:    nil,
		},
		{
			name:   "检查用户权限_无权限",
			before: func(t *testing.T) {},
			newSvcFunc: func(t *testing.T) service.Service {
				t.Helper()
				return service.NewPermissionService(s.repo)
			},
			req: domain.PersonalPermission{
				UID:   testID,
				Biz:   "NoPermission",
				BizID: 1,
			},
			wantResult: false,
			wantErr:    nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			svc := tc.newSvcFunc(t)
			actualResult, err := svc.HasPersonalPermission(context.Background(), tc.req)
			require.Equal(t, tc.wantResult, actualResult)
			require.Equal(t, tc.wantErr, err)
		})
	}
}

func (s *ModuleTestSuite) TestService_FindPersonalPermissions() {
	t := s.T()

	testCases := []struct {
		name       string
		before     func(t *testing.T)
		newSvcFunc func(t *testing.T) service.Service
		uid        int64

		wantResult map[string][]domain.PersonalPermission
		wantErr    error
	}{
		{
			name: "查找用户权限数据_找到数据并按照biz分组返回",
			before: func(t *testing.T) {
				t.Helper()
				uid := int64(79080127)
				var err = s.repo.CreatePersonalPermission(context.Background(), []domain.PersonalPermission{
					{
						UID:   uid,
						Biz:   "music",
						BizID: 52,
					},
					{
						UID:   uid,
						Biz:   "music",
						BizID: 57,
					},
					{
						UID:   uid,
						Biz:   "book",
						BizID: 52,
					},
				})
				require.NoError(t, err)
			},
			newSvcFunc: func(t *testing.T) service.Service {
				t.Helper()
				return service.NewPermissionService(s.repo)
			},
			uid: 79080127,
			wantResult: map[string][]domain.PersonalPermission{
				"music": {
					{
						UID:   79080127,
						Biz:   "music",
						BizID: 52,
					},
					{
						UID:   79080127,
						Biz:   "music",
						BizID: 57,
					},
				},
				"book": {
					{
						UID:   79080127,
						Biz:   "book",
						BizID: 52,
					},
				},
			},
			wantErr: nil,
		},
		{
			name:   "查找用户权限数据_无权限数据",
			before: func(t *testing.T) {},
			newSvcFunc: func(t *testing.T) service.Service {
				t.Helper()
				return service.NewPermissionService(s.repo)
			},
			uid:        2179832,
			wantResult: map[string][]domain.PersonalPermission{},
			wantErr:    nil,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			svc := tc.newSvcFunc(t)
			actualResult, err := svc.FindPersonalPermissions(context.Background(), tc.uid)
			require.Equal(t, tc.wantErr, err)
			require.Equal(t, tc.wantResult, actualResult)
		})
	}

}
