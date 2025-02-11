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
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/ecodeclub/webook/internal/pkg/middleware"

	"github.com/ecodeclub/webook/internal/pkg/snowflake"
	"github.com/ecodeclub/webook/internal/user"

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
		}, session.DefaultProvider(), nil)
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

type HandlerWithAppTestSuite struct {
	suite.Suite
	db            *egorm.Component
	server        *egin.Component
	mockWeSvc     *svcmocks.MockOAuth2Service
	mockWeMiniSvc *svcmocks.MockOAuth2Service
	mockPermSvc   *permissionmocks.MockService
	userSvc       user.UserService
}

func (s *HandlerWithAppTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
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
		}, session.DefaultProvider(), nil)
	s.mockWeSvc = wesvc
	s.mockWeMiniSvc = weMiniSvc
	//
	server.Use(func(ctx *gin.Context) {
		ctx.Set("_session", session.NewMemorySession(session.Claims{
			// 此id为生成的 appid为1的id
			Uid: 1814857761469632512,
		}))
	})

	server.Use(middleware.NewCheckAppIdBuilder().Build())

	hdl.PrivateRoutes(server.Engine)
	hdl.PublicRoutes(server.Engine)
	s.server = server
	m := startup.InitModule()
	s.userSvc = m.Svc
}

func (s *HandlerWithAppTestSuite) TearDownSuite() {
	err := s.db.Exec("TRUNCATE table `users`").Error
	require.NoError(s.T(), err)
	err = s.db.Exec("TRUNCATE table `users_ielts`").Error
	require.NoError(s.T(), err)
}

func (s *HandlerWithAppTestSuite) TestEditProfile() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      EditReq
		app      int64
		wantResp test.Result[any]
		wantCode int
	}{
		{
			name: "编辑成功",
			before: func(t *testing.T) {
				u := &dao.User{
					Id:       1814857761469632512,
					Nickname: "old name",
					Avatar:   "old avatar",
				}
				err := s.db.WithContext(context.Background()).Table("users_ielts").Create(u).Error
				require.NoError(t, err)
				return
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Table("users_ielts").Where("id = ?", 1814857761469632512).First(&u).Error
				require.NoError(t, err)
				u.Ctime = 0
				u.Utime = 0
				assert.Equal(t, dao.User{
					Id:       1814857761469632512,
					Avatar:   "new avatar",
					Nickname: "new name",
				}, u)
			},
			app: 1,
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
				err := s.db.Table("users_ielts").Create(&dao.User{
					Id:       1814857761469632512,
					Nickname: "old name",
					Avatar:   "old avatar",
				}).Error
				require.NoError(t, err)
			},
			app: 1,
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Table("users_ielts").Where("id = ?", 1814857761469632512).First(&u).Error
				require.NoError(t, err)
				u.Ctime = 0
				u.Utime = 0
				assert.Equal(t, dao.User{
					Id:       1814857761469632512,
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
			req.Header.Set("X-APP", fmt.Sprintf("%d", tc.app))
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
			// 清理掉 数据 的数据
			s.db.Exec("TRUNCATE table `users`")
			s.db.Exec("TRUNCATE table `users_ielts`")
		})
	}
}

func (s *HandlerWithAppTestSuite) TestEditProfileSvc() {
	testCases := []struct {
		name       string
		before     func(t *testing.T)
		after      func(t *testing.T)
		modifyUser domain.User
	}{
		{
			name: "编辑成功",
			before: func(t *testing.T) {
				u := &dao.User{
					Id:       1814857761469632512,
					Nickname: "old name",
					Avatar:   "old avatar",
				}
				err := s.db.WithContext(context.Background()).Table("users_ielts").Create(u).Error
				require.NoError(t, err)
				return
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Table("users_ielts").Where("id = ?", 1814857761469632512).First(&u).Error
				require.NoError(t, err)
				u.Ctime = 0
				u.Utime = 0
				assert.Equal(t, dao.User{
					Id:       1814857761469632512,
					Avatar:   "new avatar",
					Nickname: "new name",
				}, u)
			},
			modifyUser: domain.User{
				Id:       1814857761469632512,
				Avatar:   "new avatar",
				Nickname: "new name",
			},
		},
		{
			name: "兼容之前webook的uid。",
			before: func(t *testing.T) {
				u := &dao.User{
					Id:       123,
					Nickname: "old name",
					Avatar:   "old avatar",
				}
				err := s.db.WithContext(context.Background()).Create(u).Error
				require.NoError(t, err)
				return
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
			modifyUser: domain.User{
				Id:       123,
				Avatar:   "new avatar",
				Nickname: "new name",
			},
		},
		{
			name: "编辑成功-部分数据",
			before: func(t *testing.T) {
				err := s.db.Table("users_ielts").Create(&dao.User{
					Id:       1814857761469632512,
					Nickname: "old name",
					Avatar:   "old avatar",
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Table("users_ielts").Where("id = ?", 1814857761469632512).First(&u).Error
				require.NoError(t, err)
				u.Ctime = 0
				u.Utime = 0
				assert.Equal(t, dao.User{
					Id:       1814857761469632512,
					Avatar:   "old avatar",
					Nickname: "new name",
				}, u)
			},
			modifyUser: domain.User{
				Id:       1814857761469632512,
				Nickname: "new name",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			ctx := context.WithValue(context.Background(), "uid", tc.modifyUser.Id)
			err := s.userSvc.UpdateNonSensitiveInfo(ctx, tc.modifyUser)
			require.NoError(t, err)
			tc.after(t)
			// 清理掉 数据 的数据
			s.db.Exec("TRUNCATE table `users`")
			s.db.Exec("TRUNCATE table `users_ielts`")
		})
	}
}

func (s *HandlerWithAppTestSuite) TestVerify() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		app      int
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
								Uid:   1814857761469632512,
								Biz:   "project",
								BizID: 1234,
								Desc:  "项目权限",
							},
						},
					}, nil)
			},
			app: 1,
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Table("users_ielts").Where("wechat_union_id = ?", "mock-union-id-1").First(&u).Error
				require.NoError(t, err)
				app := snowflake.ID(u.Id).AppID()
				require.Equal(t, uint(1), app)
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
								Uid:   1814857761469632512,
								Biz:   "project",
								BizID: 1234,
								Desc:  "项目权限",
							},
						},
					}, nil)
				err := s.db.Table("users_ielts").Create(&dao.User{
					Id:               1814857761469632512,
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
				err := s.db.Table("users_ielts").Where("wechat_union_id = ?", "mock-union-id-2").First(&u).Error
				require.NoError(t, err)
				assert.Equal(t, dao.User{
					Id:               1814857761469632512,
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
			app: 1,
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
			req.Header.Set("X-APP", fmt.Sprintf("%d", tc.app))
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
			// 清理掉的数据
			err = s.db.Exec("TRUNCATE table `users`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `users_ielts`").Error
			require.NoError(t, err)
		})
	}
}

func (s *HandlerWithAppTestSuite) TestMiniVerify() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		appid    int64
		req      web.WechatCallback
		wantResp test.Result[web.Profile]
		wantCode int
	}{
		{
			name:  "创建",
			appid: 1,
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
				var u dao.User
				err := s.db.Table("users_ielts").Where("wechat_union_id = ?", "mock-union-id-1").First(&u).Error
				require.NoError(t, err)
				app := snowflake.ID(u.Id).AppID()
				require.Equal(t, uint(1), app)
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
				err := s.db.Table("users_ielts").Create(&dao.User{
					Id:            1814857761469632512,
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
			appid: 1,
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Table("users_ielts").Where("wechat_union_id = ?", "mock-union-id-2").First(&u).Error
				require.NoError(t, err)
				assert.Equal(t, dao.User{
					Id:               1814857761469632512,
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
			req.Header.Set("X-APP", fmt.Sprintf("%d", tc.appid))
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
			err = s.db.Exec("TRUNCATE table `users_ielts`").Error
			require.NoError(t, err)
		})
	}
}

func (s *HandlerWithAppTestSuite) TestProfile() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		wantResp test.Result[web.Profile]
		appid    int64
		wantCode int
	}{
		{
			name:  "获得数据",
			appid: 1,
			before: func(t *testing.T) {
				err := s.db.Table("users_ielts").Create(&dao.User{
					Id:       1814857761469632512,
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
			req.Header.Set("X-APP", fmt.Sprintf("%d", tc.appid))
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Profile]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
		})
	}
}

// 直接调用svc中的FindOrCreateByWechat方法
func (s *HandlerWithAppTestSuite) TestFindOrCreateByWechat() {
	testcases := []struct {
		name   string
		info   domain.WechatInfo
		ctx    func() context.Context
		before func(t *testing.T)
		after  func(t *testing.T)
	}{
		{
			name: "不存在创建",
			info: domain.WechatInfo{
				OpenId:  "mock_openid",
				UnionId: "mock_union_id",
			},
			before: func(t *testing.T) {},
			ctx: func() context.Context {
				ctx := context.WithValue(context.Background(), middleware.AppCtxKey, uint(1))
				return ctx
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Table("users_ielts").Where("wechat_union_id = ?", "mock_union_id").First(&u).Error
				require.NoError(t, err)
				app := snowflake.ID(u.Id).AppID()
				require.Equal(t, uint(1), app)
			},
		},
		{
			name: "更新unionId",
			info: domain.WechatInfo{
				OpenId:  "mock-open-id-2",
				UnionId: "mock-union-id-2",
			},
			before: func(t *testing.T) {
				err := s.db.Table("users_ielts").Create(&dao.User{
					Id:            1814857761469632512,
					Avatar:        "mock avatar",
					Nickname:      "nickname",
					SN:            "mock-sn-123",
					WechatUnionId: sqlx.NewNullString("mock-union-id-2"),
					Ctime:         123,
					Utime:         123,
				}).Error
				require.NoError(t, err)
			},
			ctx: func() context.Context {
				return context.WithValue(context.Background(), "uid", int64(1814857761469632512))
			},
			after: func(t *testing.T) {
				var u dao.User
				err := s.db.Table("users_ielts").Where("wechat_union_id = ?", "mock-union-id-2").First(&u).Error
				require.NoError(t, err)
				app := snowflake.ID(u.Id).AppID()
				require.Equal(t, uint(1), app)
				require.Equal(t, sql.NullString{
					Valid:  true,
					String: "mock-open-id-2",
				}, u.WechatOpenId)
			},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			ctx := tc.ctx()
			_, err := s.userSvc.FindOrCreateByWechat(ctx, tc.info)
			require.NoError(t, err)
			tc.after(t)
			// 清理掉 123 的数据
			err = s.db.Exec("TRUNCATE table `users`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `users_ielts`").Error
			require.NoError(t, err)
		})
	}

}

// 直接调用svc中的Profile方法
func (s *HandlerWithAppTestSuite) TestProfileSvc() {
	testcases := []struct {
		name   string
		info   domain.WechatInfo
		ctx    func() context.Context
		before func(t *testing.T)
		us     domain.User
	}{
		{
			name: "通过往ctx添加uid",
			info: domain.WechatInfo{},
			ctx: func() context.Context {
				return context.WithValue(context.Background(), "uid", int64(1814857761469632513))
			},
			before: func(t *testing.T) {
				err := s.db.Table("users_ielts").Create(&dao.User{
					Id:       1814857761469632513,
					SN:       "11111",
					Nickname: "old name",
					Avatar:   "old avatar",
				}).Error
				require.NoError(t, err)
			},
			us: domain.User{
				Id:       1814857761469632513,
				SN:       "11111",
				Nickname: "old name",
				Avatar:   "old avatar",
			},
		},
	}
	for _, tc := range testcases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			u, err := s.userSvc.Profile(tc.ctx(), tc.us.Id)
			require.NoError(t, err)
			assert.Equal(t, tc.us, u)
			// 清理掉 123 的数据
			err = s.db.Exec("TRUNCATE table `users`").Error
			require.NoError(t, err)
			err = s.db.Exec("TRUNCATE table `users_ielts`").Error
			require.NoError(t, err)
		})
	}

}

func TestHandlerWithApp(t *testing.T) {
	suite.Run(t, new(HandlerWithAppTestSuite))
}
