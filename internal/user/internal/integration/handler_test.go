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
	"net/http"
	"testing"

	"github.com/ecodeclub/ekit/sqlx"

	permissionmocks "github.com/ecodeclub/webook/internal/permission/mocks"
	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	svcmocks "github.com/ecodeclub/webook/internal/user/internal/service/mocks"
	"github.com/stretchr/testify/assert"

	"github.com/ecodeclub/webook/internal/member"
	membermocks "github.com/ecodeclub/webook/internal/member/mocks"
	"github.com/ecodeclub/webook/internal/permission"
	"go.uber.org/mock/gomock"

	"github.com/ecodeclub/ekit/iox"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/test"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/user/internal/integration/startup"
	"github.com/ecodeclub/webook/internal/user/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/user/internal/web"
	"github.com/ego-component/egorm"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/econf"
	"github.com/gotomicro/ego/server/egin"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HandleTestSuite struct {
	suite.Suite
	db            *egorm.Component
	server        *egin.Component
	mockWeSvc     *svcmocks.MockOAuth2Service
	mockWeMiniSvc *svcmocks.MockOAuth2Service
	mockPermSvc   *permissionmocks.MockService
}

func (s *HandleTestSuite) SetupSuite() {
	econf.Set("http_users", map[string]any{})
	s.db = testioc.InitDB()
	err := dao.InitTables(s.db)
	require.NoError(s.T(), err)
	econf.Set("server", map[string]any{"debug": true})
	server := egin.Load("server").Build()
	ctrl := gomock.NewController(s.T())
	memSvc := membermocks.NewMockService(ctrl)
	memSvc.EXPECT().GetMembershipInfo(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context,
			uid int64) (member.Member, error) {
			return member.Member{
				Uid:   uid,
				EndAt: 1234,
			}, nil
		}).AnyTimes()
	permSvc := permissionmocks.NewMockService(ctrl)
	s.mockPermSvc = permSvc
	wesvc := svcmocks.NewMockOAuth2Service(ctrl)
	weMiniSvc := svcmocks.NewMockOAuth2Service(ctrl)
	hdl := startup.InitHandler(wesvc,
		weMiniSvc,
		&member.Module{Svc: memSvc},
		&permission.Module{
			Svc: permSvc,
		}, nil)
	s.mockWeSvc = wesvc
	s.mockWeMiniSvc = weMiniSvc
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			Uid: 123,
		}))
	})

	hdl.PrivateRoutes(server.Engine)
	hdl.PublicRoutes(server.Engine)
	s.server = server
}

func (s *HandleTestSuite) TearDownSuite() {
	err := s.db.Exec("TRUNCATE table `users`").Error
	require.NoError(s.T(), err)
}

func (s *HandleTestSuite) TestEditProfile() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      EditReq
		wantResp test.Result[any]
		wantCode int
	}{
		{
			name: "编辑成功",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.User{
					Id:       123,
					Nickname: "old name",
					Avatar:   "old avatar",
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Where("id = ?", 123).First(&u).Error
				require.NoError(t, err)
				u.Ctime = 0
				u.Utime = 0
				assert.Equal(t, dao.User{
					Id:       123,
					Avatar:   "new avatar",
					Nickname: "new name",
				}, u)
			},
			req: EditReq{
				Avatar:   "new avatar",
				Nickname: "new name",
			},
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			wantCode: 200,
		},
		{
			name: "编辑成功-部分数据",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.User{
					Id:       123,
					Nickname: "old name",
					Avatar:   "old avatar",
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Where("id = ?", 123).First(&u).Error
				require.NoError(t, err)
				u.Ctime = 0
				u.Utime = 0
				assert.Equal(t, dao.User{
					Id:       123,
					Avatar:   "old avatar",
					Nickname: "new name",
				}, u)
			},
			req: EditReq{
				Nickname: "new name",
			},
			wantResp: test.Result[any]{
				Msg: "OK",
			},
			wantCode: 200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/users/profile", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			// 清理掉 123 的数据
			s.db.Exec("TRUNCATE table `users`")
		})
	}
}

func (s *HandleTestSuite) TestVerify() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.WechatCallback
		wantResp test.Result[web.Profile]
		wantCode int
	}{
		{
			name: "创建",
			before: func(t *testing.T) {
				// 设置邀请码
				s.mockWeSvc.EXPECT().Verify(gomock.Any(), service.CallbackParams{
					State: "mock state 1",
					Code:  "wechat code 1",
				}).Return(domain.WechatInfo{
					OpenId:         "mock-open-id-1",
					UnionId:        "mock-union-id-1",
					InvitationCode: "invitation-code",
				}, nil)
				s.mockPermSvc.EXPECT().
					FindPersonalPermissions(gomock.Any(), gomock.Any()).
					Return(map[string][]permission.Permission{
						"project": {
							{
								Uid:   123,
								Biz:   "project",
								BizID: 1234,
								Desc:  "项目权限",
							},
						},
					}, nil)
			},
			after: func(t *testing.T) {

			},
			req: web.WechatCallback{
				State: "mock state 1",
				Code:  "wechat code 1",
			},
			wantResp: test.Result[web.Profile]{
				Data: web.Profile{
					MemberDDL: 1234,
				},
			},
			wantCode: 200,
		},
		{
			name: "查找-更新 OpenId",
			before: func(t *testing.T) {
				// 设置邀请码
				s.mockWeSvc.EXPECT().Verify(gomock.Any(), service.CallbackParams{
					State: "mock state 2",
					Code:  "wechat code 2",
				}).Return(domain.WechatInfo{
					OpenId:         "mock-open-id-2",
					UnionId:        "mock-union-id-2",
					InvitationCode: "invitation-code",
				}, nil)
				s.mockPermSvc.EXPECT().
					FindPersonalPermissions(gomock.Any(), gomock.Any()).
					Return(map[string][]permission.Permission{
						"project": {
							{
								Uid:   123,
								Biz:   "project",
								BizID: 1234,
								Desc:  "项目权限",
							},
						},
					}, nil)
				err := s.db.Create(&dao.User{
					Id:               123,
					Avatar:           "mock avatar",
					Nickname:         "nickname",
					SN:               "mock-sn-123",
					WechatUnionId:    sqlx.NewNullString("mock-union-id-2"),
					WechatMiniOpenId: sqlx.NewNullString("mock-mini-open-id-2"),
					Ctime:            123,
					Utime:            123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Where("wechat_union_id = ?", "mock-union-id-2").First(&u).Error
				require.NoError(t, err)
				assert.Equal(t, dao.User{
					Id:               123,
					Avatar:           "mock avatar",
					Nickname:         "nickname",
					SN:               "mock-sn-123",
					WechatUnionId:    sqlx.NewNullString("mock-union-id-2"),
					WechatOpenId:     sqlx.NewNullString("mock-open-id-2"),
					WechatMiniOpenId: sqlx.NewNullString("mock-mini-open-id-2"),
					Ctime:            123,
					Utime:            123,
				}, u)
			},
			req: web.WechatCallback{
				State: "mock state 2",
				Code:  "wechat code 2",
			},
			wantResp: test.Result[web.Profile]{
				Data: web.Profile{
					MemberDDL: 1234,
					Avatar:    "mock avatar",
				},
			},
			wantCode: 200,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/oauth2/wechat/callback", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Profile]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			val := recorder.MustScan()
			assert.NotEmpty(t, val.Data.SN)
			val.Data.SN = ""
			assert.NotEmpty(t, val.Data.Nickname)
			// 在创建的时候，是随机生成的昵称，所以需要特殊判断
			val.Data.Nickname = ""
			assert.Equal(t, tc.wantResp, val)
			tc.after(t)
			// 清理掉 123 的数据
			err = s.db.Exec("TRUNCATE table `users`").Error
			require.NoError(t, err)
		})
	}
}

func (s *HandleTestSuite) TestMiniVerify() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		req      web.WechatCallback
		wantResp test.Result[web.Profile]
		wantCode int
	}{
		{
			name: "创建",
			before: func(t *testing.T) {
				// 设置邀请码
				s.mockWeMiniSvc.EXPECT().Verify(gomock.Any(), service.CallbackParams{
					State: "mock state 1",
					Code:  "wechat code 1",
				}).Return(domain.WechatInfo{
					OpenId:         "mock-open-id-1",
					UnionId:        "mock-union-id-1",
					InvitationCode: "invitation-code",
				}, nil)
				s.mockPermSvc.EXPECT().
					FindPersonalPermissions(gomock.Any(), gomock.Any()).
					Return(map[string][]permission.Permission{
						"project": {
							{
								Uid:   123,
								Biz:   "project",
								BizID: 1234,
								Desc:  "项目权限",
							},
						},
					}, nil)
			},
			after: func(t *testing.T) {

			},
			req: web.WechatCallback{
				State: "mock state 1",
				Code:  "wechat code 1",
			},
			wantResp: test.Result[web.Profile]{
				Data: web.Profile{
					MemberDDL: 1234,
				},
			},
			wantCode: 200,
		},
		{
			name: "查找-更新 OpenId",
			before: func(t *testing.T) {
				// 设置邀请码
				s.mockWeMiniSvc.EXPECT().Verify(gomock.Any(), service.CallbackParams{
					State: "mock state 2",
					Code:  "wechat code 2",
				}).Return(domain.WechatInfo{
					UnionId:        "mock-union-id-2",
					MiniOpenId:     "mock-mini-open-id-2",
					InvitationCode: "invitation-code",
				}, nil)
				s.mockPermSvc.EXPECT().
					FindPersonalPermissions(gomock.Any(), gomock.Any()).
					Return(map[string][]permission.Permission{
						"project": {
							{
								Uid:   123,
								Biz:   "project",
								BizID: 1234,
								Desc:  "项目权限",
							},
						},
					}, nil)
				err := s.db.Create(&dao.User{
					Id:            123,
					Avatar:        "mock avatar",
					Nickname:      "nickname",
					SN:            "mock-sn-123",
					WechatUnionId: sqlx.NewNullString("mock-union-id-2"),
					WechatOpenId:  sqlx.NewNullString("mock-open-id-2"),
					Ctime:         123,
					Utime:         123,
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Where("wechat_union_id = ?", "mock-union-id-2").First(&u).Error
				require.NoError(t, err)
				assert.Equal(t, dao.User{
					Id:               123,
					Avatar:           "mock avatar",
					Nickname:         "nickname",
					SN:               "mock-sn-123",
					WechatUnionId:    sqlx.NewNullString("mock-union-id-2"),
					WechatOpenId:     sqlx.NewNullString("mock-open-id-2"),
					WechatMiniOpenId: sqlx.NewNullString("mock-mini-open-id-2"),
					Ctime:            123,
					Utime:            123,
				}, u)
			},
			req: web.WechatCallback{
				State: "mock state 2",
				Code:  "wechat code 2",
			},
			wantResp: test.Result[web.Profile]{
				Data: web.Profile{
					MemberDDL: 1234,
					Avatar:    "mock avatar",
				},
			},
			wantCode: 200,
		},
	}
	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/oauth2/wechat/mini/callback", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Profile]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			val := recorder.MustScan()
			assert.NotEmpty(t, val.Data.SN)
			val.Data.SN = ""
			assert.NotEmpty(t, val.Data.Nickname)
			// 在创建的时候，是随机生成的昵称，所以需要特殊判断
			val.Data.Nickname = ""
			assert.Equal(t, tc.wantResp, val)
			tc.after(t)
			// 清理掉 123 的数据
			err = s.db.Exec("TRUNCATE table `users`").Error
			require.NoError(t, err)
		})
	}
}

func (s *HandleTestSuite) TestProfile() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		wantResp test.Result[web.Profile]
		wantCode int
	}{
		{
			name: "获得数据",
			before: func(t *testing.T) {
				err := s.db.Create(&dao.User{
					Id:       123,
					Nickname: "old name",
					Avatar:   "old avatar",
				}).Error
				require.NoError(t, err)
			},
			wantResp: test.Result[web.Profile]{
				Data: web.Profile{
					Nickname:  "old name",
					Avatar:    "old avatar",
					MemberDDL: 1234,
				},
			},
			wantCode: 200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodGet,
				"/users/profile", nil)
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Profile]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

func TestUserHandler(t *testing.T) {
	suite.Run(t, new(HandleTestSuite))
}

type EditReq struct {
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}
