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

package middleware

import (
	"errors"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	membermocks "github.com/ecodeclub/webook/internal/member/mocks"
	"github.com/stretchr/testify/assert"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/member"
	sessmocks "github.com/ecodeclub/webook/internal/test/mocks"
	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func TestCheck(t *testing.T) {
	testCases := []struct {
		name      string
		mock      func(ctrl *gomock.Controller) (member.Service, session.Provider)
		wantCode  int
		afterFunc func(t *testing.T, ctx *ginx.Context)
	}{
		{
			name: "无JWT",
			mock: func(ctrl *gomock.Controller) (member.Service, session.Provider) {
				mockP := sessmocks.NewMockProvider(ctrl)
				mockP.EXPECT().Get(gomock.Any()).Return(nil, errors.New("mock no jwt"))
				return nil, mockP
			},
			afterFunc: func(t *testing.T, ctx *ginx.Context) {},
			wantCode:  403,
		},
		{
			// 应该成功_JWT有会员截止日期_会员生效中
			name: "JWT有效会员",
			mock: func(ctrl *gomock.Controller) (member.Service, session.Provider) {
				mockP := sessmocks.NewMockProvider(ctrl)
				claims := session.Claims{
					Uid:  2793,
					SSID: "ssid-2793",
					Data: map[string]string{
						"memberDDL": strconv.FormatInt(time.Now().Add(1*24*time.Hour).UnixMilli(), 10),
					},
				}
				mockSession := sessmocks.NewMockSession(ctrl)
				mockSession.EXPECT().Claims().Return(claims)
				mockP.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				return nil, mockP
			},
			afterFunc: func(t *testing.T, ctx *ginx.Context) {},
			wantCode:  200,
		},
		{
			name: "JWT会员过期-续费",
			mock: func(ctrl *gomock.Controller) (member.Service, session.Provider) {
				service := membermocks.NewMockService(ctrl)
				newExpired := time.Now().Add(time.Hour)
				service.EXPECT().GetMembershipInfo(gomock.Any(), int64(2795)).
					Return(member.Member{
						Uid:   2795,
						EndAt: newExpired.UnixMilli(),
					}, nil)

				mockSession := sessmocks.NewMockSession(ctrl)
				expired := strconv.FormatInt(time.Now().Add(-1*24*time.Hour).UnixMilli(), 10)
				claims := session.Claims{
					Uid:  2795,
					SSID: "ssid-2795",
					// 模拟过期了
					Data: map[string]string{
						"memberDDL": expired,
					},
				}
				mockSession.EXPECT().Claims().Return(claims)
				provider := sessmocks.NewMockProvider(ctrl)
				provider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)
				provider.EXPECT().UpdateClaims(gomock.Any(), session.Claims{
					Uid:  2795,
					SSID: "ssid-2795",
					Data: map[string]string{
						"memberDDL": strconv.FormatInt(newExpired.UnixMilli(), 10),
					},
				}).Return(nil)
				return service, provider
			},
			afterFunc: func(t *testing.T, ctx *ginx.Context) {},
			wantCode:  200,
		},
		{
			name: "JWT会员过期-会员查找失败",
			mock: func(ctrl *gomock.Controller) (member.Service, session.Provider) {
				service := membermocks.NewMockService(ctrl)
				service.EXPECT().GetMembershipInfo(gomock.Any(), int64(2795)).
					Return(member.Member{}, errors.New("mock error"))

				mockSession := sessmocks.NewMockSession(ctrl)
				expired := strconv.FormatInt(time.Now().Add(-1*24*time.Hour).UnixMilli(), 10)
				claims := session.Claims{
					Uid:  2795,
					SSID: "ssid-2795",
					// 模拟过期了
					Data: map[string]string{
						"memberDDL": expired,
					},
				}
				mockSession.EXPECT().Claims().Return(claims)
				provider := sessmocks.NewMockProvider(ctrl)
				provider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)
				return service, provider
			},
			afterFunc: func(t *testing.T, ctx *ginx.Context) {},
			wantCode:  403,
		},

		{
			name: "JWT会员过期-全过期",
			mock: func(ctrl *gomock.Controller) (member.Service, session.Provider) {
				service := membermocks.NewMockService(ctrl)
				newExpired := time.Now().Add(-time.Hour)
				service.EXPECT().GetMembershipInfo(gomock.Any(), int64(2795)).
					Return(member.Member{
						Uid:   2795,
						EndAt: newExpired.UnixMilli(),
					}, nil)

				mockSession := sessmocks.NewMockSession(ctrl)
				expired := strconv.FormatInt(time.Now().Add(-1*24*time.Hour).UnixMilli(), 10)
				claims := session.Claims{
					Uid:  2795,
					SSID: "ssid-2795",
					// 模拟过期了
					Data: map[string]string{
						"memberDDL": expired,
					},
				}
				mockSession.EXPECT().Claims().Return(claims)
				provider := sessmocks.NewMockProvider(ctrl)
				provider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)
				return service, provider
			},
			afterFunc: func(t *testing.T, ctx *ginx.Context) {},
			wantCode:  403,
		},

		{
			name: "JWT会员过期-刷新token失败",
			mock: func(ctrl *gomock.Controller) (member.Service, session.Provider) {
				service := membermocks.NewMockService(ctrl)
				newExpired := time.Now().Add(time.Hour)
				service.EXPECT().GetMembershipInfo(gomock.Any(), int64(2795)).
					Return(member.Member{
						Uid:   2795,
						EndAt: newExpired.UnixMilli(),
					}, nil)

				mockSession := sessmocks.NewMockSession(ctrl)
				expired := strconv.FormatInt(time.Now().Add(-1*24*time.Hour).UnixMilli(), 10)
				claims := session.Claims{
					Uid:  2795,
					SSID: "ssid-2795",
					// 模拟过期了
					Data: map[string]string{
						"memberDDL": expired,
					},
				}
				mockSession.EXPECT().Claims().Return(claims)
				provider := sessmocks.NewMockProvider(ctrl)
				provider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)
				provider.EXPECT().UpdateClaims(gomock.Any(), session.Claims{
					Uid:  2795,
					SSID: "ssid-2795",
					Data: map[string]string{
						"memberDDL": strconv.FormatInt(newExpired.UnixMilli(), 10),
					},
				}).Return(errors.New("mock error"))
				return service, provider
			},
			afterFunc: func(t *testing.T, ctx *ginx.Context) {},
			wantCode:  403,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			svc, p := tc.mock(ctrl)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			builder := NewCheckMembershipMiddlewareBuilder(svc)
			builder.sp = p
			hdl := builder.Build()
			hdl(c)
			assert.Equal(t, tc.wantCode, c.Writer.Status())
		})
	}
}
