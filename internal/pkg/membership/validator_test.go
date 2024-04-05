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

package membership

import (
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/member"
	membermocks "github.com/ecodeclub/webook/internal/member/mocks"
	sessmocks "github.com/ecodeclub/webook/internal/pkg/membership/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestNewValidator(t *testing.T) {

	testCases := map[string]struct {
		svcFunc        func(ctrl *gomock.Controller) member.Service
		sessFunc       func(ctrl *gomock.Controller) session.Session
		requireErrFunc require.ErrorAssertionFunc
		wantResult     ginx.Result
		afterFunc      func(t *testing.T, ctx *ginx.Context)
	}{
		"应该成功_JWT有会员截止日期_会员生效中": {
			svcFunc: func(ctrl *gomock.Controller) member.Service {
				return nil
			},
			sessFunc: func(ctrl *gomock.Controller) session.Session {
				mockSession := sessmocks.NewMockSession(ctrl)
				claims := session.Claims{
					Uid:  2793,
					SSID: "ssid-2793",
					Data: map[string]string{
						"memberDDL": time.Now().Add(1 * 24 * time.Hour).Local().Format(time.DateTime),
					},
				}
				mockSession.EXPECT().Claims().Return(claims)
				return mockSession
			},
			requireErrFunc: require.NoError,
			afterFunc:      func(t *testing.T, ctx *ginx.Context) {},
		},

		"应该失败_JWT有会员截止日期_会员已过期": {
			svcFunc: func(ctrl *gomock.Controller) member.Service {
				return nil
			},
			sessFunc: func(ctrl *gomock.Controller) session.Session {
				mockSession := sessmocks.NewMockSession(ctrl)
				claims := session.Claims{
					Uid:  2794,
					SSID: "ssid-2794",
					Data: map[string]string{
						"memberDDL": time.Now().Add(-1 * 24 * time.Hour).Local().Format(time.DateTime),
					},
				}
				mockSession.EXPECT().Claims().Return(claims)
				return mockSession
			},
			requireErrFunc: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorIs(t, err, ErrMembershipExpired)
			},
			afterFunc: func(t *testing.T, ctx *ginx.Context) {},
		},

		"应该失败_JWT无会员截止日期_未找到会员信息": {
			svcFunc: func(ctrl *gomock.Controller) member.Service {
				service := membermocks.NewMockService(ctrl)
				service.EXPECT().GetMembershipInfo(gomock.Any(), int64(2795)).Return(member.Member{}, errors.New("模拟会员信息未找到错误"))
				return service
			},
			sessFunc: func(ctrl *gomock.Controller) session.Session {
				mockSession := sessmocks.NewMockSession(ctrl)
				claims := session.Claims{
					Uid:  2795,
					SSID: "ssid-2795",
					Data: map[string]string{},
				}
				mockSession.EXPECT().Claims().Return(claims)
				return mockSession
			},
			requireErrFunc: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorIs(t, err, ErrGetMemberInfo)
			},
			afterFunc: func(t *testing.T, ctx *ginx.Context) {},
		},

		"应该失败_JWT无会员截止日期_找到会员信息_会员已过期": {
			svcFunc: func(ctrl *gomock.Controller) member.Service {
				service := membermocks.NewMockService(ctrl)
				service.EXPECT().GetMembershipInfo(gomock.Any(), int64(2796)).Return(member.Member{
					UID:     2796,
					StartAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local).Unix(),
				}, nil)
				return service
			},
			sessFunc: func(ctrl *gomock.Controller) session.Session {
				mockSession := sessmocks.NewMockSession(ctrl)
				claims := session.Claims{
					Uid:  2796,
					SSID: "ssid-2796",
					Data: map[string]string{},
				}
				mockSession.EXPECT().Claims().Return(claims)
				return mockSession
			},
			requireErrFunc: func(t require.TestingT, err error, i ...interface{}) {
				require.ErrorIs(t, err, ErrMembershipExpired)
			},
			afterFunc: func(t *testing.T, ctx *ginx.Context) {},
		},
		// "应该成功_无会员截止日期_找到会员信息_会员生效中": {
		//
		// 	svcFunc: func(ctrl *gomock.Controller) member.Service {
		//
		// 		return nil
		// 	},
		// 	sessFunc: func(ctrl *gomock.Controller) session.Session {
		// 		mockSession := sessmocks.NewMockSession(ctrl)
		// 		claims := session.Claims{
		// 			Uid:  2795,
		// 			SSID: "ssid-2795",
		// 			Data: map[string]string{},
		// 		}
		// 		mockSession.EXPECT().Claims().Return(claims)
		// 		return mockSession
		// 	},
		// 	requireErrFunc: require.NoError,
		// 	afterFunc: func(t *testing.T, ctx *ginx.Context) {
		// 		sess, err := session.Get(ctx)
		// 		require.NoError(t, err)
		//
		// 		require.Equal(t, session.Claims{
		// 			Uid:  0,
		// 			SSID: "",
		// 			Data: nil,
		// 		}, sess.Claims())
		//
		// 	},
		// },
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			validator := NewValidator(tc.svcFunc(ctrl))
			res, err := validator.Membership(&ginx.Context{Context: c}, tc.sessFunc(ctrl))
			tc.requireErrFunc(t, err)
			require.Equal(t, tc.wantResult, res)
		})
	}
}
