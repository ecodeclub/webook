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
	"strings"
	"testing"

	"github.com/ecodeclub/webook/internal/user/internal/repository/cache"

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
	cache         cache.VerificationCodeCache
	server        *egin.Component
	mockWeSvc     *svcmocks.MockOAuth2Service
	mockWeMiniSvc *svcmocks.MockOAuth2Service
	mockPermSvc   *permissionmocks.MockService
}

func (s *HandleTestSuite) SetupSuite() {
	s.db = testioc.InitDB()
	ca := testioc.InitCache()
	s.cache = cache.NewVerificationCodeCache(ca)
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
		path := ctx.FullPath()
		if strings.Contains(path, "login") {
			return
		}
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
			require.True(t, val.Data.Id > 0)
			val.Data.Id = 0
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
			val.Data.Id = 0
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
					Id:        123,
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

func (s *HandleTestSuite) TestPhoneLogin() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.PhoneReq
		wantResp test.Result[web.Profile]
		wantCode int
	}{
		{
			name: "登录成功",
			before: func(t *testing.T) {
				// 创建用户
				err := s.db.Create(&dao.User{
					Id:       124,
					Nickname: "test user",
					Avatar:   "test avatar",
					Phone:    sqlx.NewNullString("13812345678"),
				}).Error
				require.NoError(t, err)
				err = s.cache.SetPhoneCode(s.T().Context(), "13812345678", "123456")
				require.NoError(t, err)
				s.mockPermSvc.EXPECT().
					FindPersonalPermissions(gomock.Any(), gomock.Any()).
					Return(map[string][]permission.Permission{
						"project": {
							{
								Uid:   124,
								Biz:   "project",
								BizID: 1234,
								Desc:  "项目权限",
							},
						},
					}, nil)

			},
			after: func(t *testing.T) {
				// 清理数据
				s.db.Exec("DELETE FROM users WHERE id = 124")
			},
			req: web.PhoneReq{
				Phone: "13812345678",
				Code:  "123456",
			},
			wantResp: test.Result[web.Profile]{
				Data: web.Profile{
					Id:        124,
					Phone:     "138****5678",
					Nickname:  "test user",
					Avatar:    "test avatar",
					MemberDDL: 1234,
				},
			},
			wantCode: 200,
		},

		{
			name: "验证码错误",
			before: func(t *testing.T) {
				// 创建用户
				err := s.db.Create(&dao.User{
					Id:       125,
					Nickname: "test user",
					Avatar:   "test avatar",
					Phone:    sqlx.NewNullString("13812345679"),
				}).Error
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 清理数据
				s.db.Exec("DELETE FROM users WHERE id = 125")
			},
			req: web.PhoneReq{
				Phone: "13812345679",
				Code:  "100006", // 错误的验证码
			},
			wantCode: 500,
			wantResp: test.Result[web.Profile]{
				Code: 501002,
				Msg:  "验证码错误",
			},
		},

		{
			name: "手机号未注册",
			before: func(t *testing.T) {
				// 模拟验证码服务返回正确验证码
				err := s.cache.SetPhoneCode(s.T().Context(), "18248862099", "234567")
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 无需清理
			},
			req: web.PhoneReq{
				Phone: "18248862099", // 未注册的手机号
				Code:  "234567",
			},
			wantResp: test.Result[web.Profile]{
				Code: 501003,
				Msg:  "手机号不存在",
			},
			wantCode: 500,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/oauth2/phone/login", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Profile]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			val := recorder.MustScan()
			if val.Data.SN != "" {
				val.Data.SN = ""
			}
			assert.Equal(t, tc.wantResp, val)
			tc.after(t)
		})
	}
}

func (s *HandleTestSuite) TestPhoneRegister() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.PhoneReq
		wantResp test.Result[web.Profile]
		wantCode int
	}{
		{
			name: "注册成功",
			before: func(t *testing.T) {
				// 设置验证码
				err := s.cache.SetPhoneCode(context.Background(), "13912345678", "123456")
				require.NoError(t, err)

				// 模拟权限服务
				s.mockPermSvc.EXPECT().
					FindPersonalPermissions(gomock.Any(), gomock.Any()).
					Return(map[string][]permission.Permission{
						"project": {
							{
								Biz:   "project",
								BizID: 1234,
								Desc:  "项目权限",
							},
						},
					}, nil)
			},
			after: func(t *testing.T) {
				// 清理创建的用户
				s.db.Exec("DELETE FROM users WHERE phone = '13912345678'")
			},
			req: web.PhoneReq{
				Phone: "13912345678",
				Code:  "123456",
			},
			wantResp: test.Result[web.Profile]{
				Data: web.Profile{
					MemberDDL: 1234,
					Phone:     "139****5678",
				},
			},
			wantCode: 200,
		},
		{
			name: "验证码错误",
			before: func(t *testing.T) {
				// 设置正确的验证码
				err := s.cache.SetPhoneCode(context.Background(), "13912345679", "654321")
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 无需清理
			},
			req: web.PhoneReq{
				Phone: "13912345679",
				Code:  "123456", // 错误的验证码
			},
			wantResp: test.Result[web.Profile]{
				Code: 501002,
				Msg:  "验证码错误",
			},
			wantCode: 500,
		},
		{
			name: "手机号已注册",
			before: func(t *testing.T) {
				// 创建已存在的用户
				err := s.db.Create(&dao.User{
					Id:       126,
					Nickname: "existing user",
					Phone:    sqlx.NewNullString("13912345681"),
				}).Error
				require.NoError(t, err)

				// 设置验证码
				err = s.cache.SetPhoneCode(context.Background(), "13912345681", "123456")
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 清理数据
				s.db.Exec("DELETE FROM users WHERE id = 126")
			},
			req: web.PhoneReq{
				Phone: "13912345681",
				Code:  "123456",
			},
			wantResp: test.Result[web.Profile]{
				Code: 501001,
				Msg:  "系统错误",
			},
			wantCode: 500,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/oauth2/phone/register", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[web.Profile]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			val := recorder.MustScan()
			// 注册成功时，SN和Nickname是随机生成的，需要忽略
			if val.Data.SN != "" {
				val.Data.SN = ""
			}
			if strings.HasPrefix(val.Data.Nickname, "用户") {
				val.Data.Nickname = ""
			}

			val.Data.Id = 0
			assert.Equal(t, tc.wantResp, val)
			tc.after(t)
		})
	}
}

func (s *HandleTestSuite) TestBindPhone() {
	testCases := []struct {
		name     string
		before   func(t *testing.T)
		after    func(t *testing.T)
		req      web.PhoneReq
		wantResp test.Result[any]
		wantCode int
	}{
		{
			name: "绑定手机号成功",
			before: func(t *testing.T) {
				// 创建用户（未绑定手机号）
				err := s.db.Create(&dao.User{
					Id:       123,
					Nickname: "test user",
					Avatar:   "test avatar",
				}).Error
				require.NoError(t, err)

				// 设置验证码
				err = s.cache.SetPhoneCode(context.Background(), "13912345678", "123456")
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 验证手机号已绑定
				var u dao.User
				err := s.db.Where("id = ?", 123).First(&u).Error
				require.NoError(t, err)
				assert.Equal(t, "13912345678", u.Phone.String)

				// 清理数据
				s.db.Exec("DELETE FROM users WHERE id = 123")
			},
			req: web.PhoneReq{
				Phone: "13912345678",
				Code:  "123456",
			},
			wantResp: test.Result[any]{},
			wantCode: 200,
		},
		{
			name: "验证码错误",
			before: func(t *testing.T) {
				// 创建用户
				err := s.db.Create(&dao.User{
					Id:       123,
					Nickname: "test user",
				}).Error
				require.NoError(t, err)

				// 设置正确的验证码
				err = s.cache.SetPhoneCode(context.Background(), "13912345679", "654321")
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 验证手机号未绑定
				var u dao.User
				err := s.db.Where("id = ?", 123).First(&u).Error
				require.NoError(t, err)
				assert.False(t, u.Phone.Valid) // 手机号应该未绑定

				// 清理数据
				s.db.Exec("DELETE FROM users WHERE id = 123")
			},
			req: web.PhoneReq{
				Phone: "13912345679",
				Code:  "123456", // 错误的验证码
			},
			wantResp: test.Result[any]{
				Code: 501002,
				Msg:  "验证码错误",
			},
			wantCode: 500,
		},
		{
			name: "验证码不存在",
			before: func(t *testing.T) {
				// 创建用户
				err := s.db.Create(&dao.User{
					Id:       123,
					Nickname: "test user",
				}).Error
				require.NoError(t, err)
				// 不设置验证码
			},
			after: func(t *testing.T) {
				// 验证手机号未绑定
				var u dao.User
				err := s.db.Where("id = ?", 123).First(&u).Error
				require.NoError(t, err)
				assert.False(t, u.Phone.Valid)

				// 清理数据
				s.db.Exec("DELETE FROM users WHERE id = 123")
			},
			req: web.PhoneReq{
				Phone: "13912345680",
				Code:  "123456",
			},
			wantResp: test.Result[any]{
				Code: 501002,
				Msg:  "验证码错误",
			},
			wantCode: 500,
		},
		{
			name: "更新手机号",
			before: func(t *testing.T) {
				// 创建已有手机号的用户
				err := s.db.Create(&dao.User{
					Id:       123,
					Nickname: "test user",
					Phone:    sqlx.NewNullString("13812345678"),
				}).Error
				require.NoError(t, err)

				// 设置验证码
				err = s.cache.SetPhoneCode(context.Background(), "13912345681", "123456")
				require.NoError(t, err)
			},
			after: func(t *testing.T) {
				// 验证手机号已更新
				var u dao.User
				err := s.db.Where("id = ?", 123).First(&u).Error
				require.NoError(t, err)
				assert.Equal(t, "13912345681", u.Phone.String)

				// 清理数据
				s.db.Exec("DELETE FROM users WHERE id = 123")
			},
			req: web.PhoneReq{
				Phone: "13912345681",
				Code:  "123456",
			},
			wantResp: test.Result[any]{},
			wantCode: 200,
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodPost,
				"/users/phone/bind", iox.NewJSONReader(tc.req))
			req.Header.Set("content-type", "application/json")
			require.NoError(t, err)
			recorder := test.NewJSONResponseRecorder[any]()
			s.server.ServeHTTP(recorder, req)
			assert.Equal(t, tc.wantCode, recorder.Code)
			assert.Equal(t, tc.wantResp, recorder.MustScan())
			tc.after(t)
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
			val.Data.Id = 0
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
			val.Data.Id = 0
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
			data := recorder.MustScan()
			data.Data.Id = 0
			assert.Equal(t, tc.wantResp, data)
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
