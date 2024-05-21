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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/ecodeclub/ekit"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/permission"
	permissionmocks "github.com/ecodeclub/webook/internal/permission/mocks"
	sessmocks "github.com/ecodeclub/webook/internal/test/mocks"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCheckPermission(t *testing.T) {
	testCases := []struct {
		name     string
		before   func(t *testing.T, ctx *gin.Context)
		mock     func(ctrl *gomock.Controller) (permission.Service, session.Provider)
		wantCode int
	}{
		{
			name: "session中有资源列表_列表包含资源_检查通过",
			before: func(t *testing.T, ctx *gin.Context) {
				t.Helper()

				bizKey := "X-Biz"
				bizIDKey := "X-Biz-ID"
				bizVal := "project"
				bizIDVal := "102"

				header := make(http.Header)
				header.Set(bizKey, bizVal)
				header.Set(bizIDKey, bizIDVal)

				ctx.Request = httptest.NewRequest(http.MethodPost, "/product/sku/detail", nil)
				ctx.Request.Header = header

				require.Equal(t, bizVal, ctx.GetHeader(bizKey))
				require.Equal(t, bizIDVal, ctx.GetHeader(bizIDKey))
			},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {

				biz := "project"
				mockProvider := sessmocks.NewMockProvider(ctrl)
				mockSession := sessmocks.NewMockSession(ctrl)

				mockSession.EXPECT().Get(gomock.Any(), biz).Return(ekit.AnyValue{Val: "103,101,102"})

				mockProvider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				return nil, mockProvider
			},
			wantCode: 200,
		},
		{
			name: "session中有资源列表_列表不含资源_实时查询有权限_更新session成功",
			before: func(t *testing.T, ctx *gin.Context) {
				t.Helper()

				bizKey := "X-Biz"
				bizIDKey := "X-Biz-ID"
				bizVal := "project"
				bizIDVal := "105"

				header := make(http.Header)
				header.Set(bizKey, bizVal)
				header.Set(bizIDKey, bizIDVal)

				ctx.Request = httptest.NewRequest(http.MethodPost, "/product/sku/detail", nil)
				ctx.Request.Header = header
				require.Equal(t, bizVal, ctx.GetHeader(bizKey))
				require.Equal(t, bizIDVal, ctx.GetHeader(bizIDKey))
			},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {

				uid := int64(2666)
				biz := "project"
				bizId := int64(105)

				mockProvider := sessmocks.NewMockProvider(ctrl)
				mockSession := sessmocks.NewMockSession(ctrl)

				mockSession.EXPECT().Get(gomock.Any(), biz).Return(ekit.AnyValue{Val: "103,101,102"})

				claims := session.Claims{
					Uid:  uid,
					SSID: "ssid-2666",
					Data: map[string]string{
						"memberDDL": strconv.FormatInt(time.Now().Add(1*24*time.Hour).UnixMilli(), 10),
					},
				}
				mockSession.EXPECT().Claims().Return(claims)
				mockSession.EXPECT().Set(gomock.Any(), biz, "103,101,102,105").Return(nil)

				mockProvider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				mockPermissionSvc := permissionmocks.NewMockService(ctrl)

				mockPermissionSvc.EXPECT().HasPermission(gomock.Any(), permission.PersonalPermission{
					Uid:   uid,
					Biz:   biz,
					BizID: bizId,
				}).Return(true, nil)

				return mockPermissionSvc, mockProvider
			},
			wantCode: 200,
		},
		{
			name: "session中无资源列表_实时查询有权限_更新session成功",
			before: func(t *testing.T, ctx *gin.Context) {
				t.Helper()

				bizKey := "X-Biz"
				bizIDKey := "X-Biz-ID"
				bizVal := "project"
				bizIDVal := "106"

				header := make(http.Header)
				header.Set(bizKey, bizVal)
				header.Set(bizIDKey, bizIDVal)

				ctx.Request = httptest.NewRequest(http.MethodPost, "/product/sku/detail", nil)
				ctx.Request.Header = header
				require.Equal(t, bizVal, ctx.GetHeader(bizKey))
				require.Equal(t, bizIDVal, ctx.GetHeader(bizIDKey))
			},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {

				uid := int64(2666)
				biz := "project"
				bizId := int64(106)

				mockProvider := sessmocks.NewMockProvider(ctrl)
				mockSession := sessmocks.NewMockSession(ctrl)

				mockSession.EXPECT().Get(gomock.Any(), biz).Return(ekit.AnyValue{Err: fmt.Errorf("%w", redis.Nil)})

				claims := session.Claims{
					Uid:  uid,
					SSID: "ssid-2666",
					Data: map[string]string{
						"memberDDL": strconv.FormatInt(time.Now().Add(1*24*time.Hour).UnixMilli(), 10),
					},
				}
				mockSession.EXPECT().Claims().Return(claims)
				mockSession.EXPECT().Set(gomock.Any(), biz, "106").Return(nil)

				mockProvider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				mockPermissionSvc := permissionmocks.NewMockService(ctrl)

				mockPermissionSvc.EXPECT().HasPermission(gomock.Any(), permission.PersonalPermission{
					Uid:   uid,
					Biz:   biz,
					BizID: bizId,
				}).Return(true, nil)

				return mockPermissionSvc, mockProvider
			},
			wantCode: 200,
		},
		{
			name: "无X-Biz请求头",
			before: func(t *testing.T, ctx *gin.Context) {
				t.Helper()

				bizKey := "X-Biz-Y"
				bizIDKey := "X-Biz-ID"
				bizVal := "project"
				bizIDVal := "110"

				header := make(http.Header)
				header.Set(bizKey, bizVal)
				header.Set(bizIDKey, bizIDVal)

				ctx.Request = httptest.NewRequest(http.MethodPost, "/product/sku/detail", nil)
				ctx.Request.Header = header
				require.Equal(t, bizVal, ctx.GetHeader(bizKey))
				require.Equal(t, bizIDVal, ctx.GetHeader(bizIDKey))
			},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {

				mockProvider := sessmocks.NewMockProvider(ctrl)
				mockSession := sessmocks.NewMockSession(ctrl)

				mockProvider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				return nil, mockProvider
			},
			wantCode: 403,
		},
		{
			name: "从session中获取资源列表失败",
			before: func(t *testing.T, ctx *gin.Context) {
				t.Helper()

				bizKey := "X-Biz"
				bizIDKey := "X-Biz-ID"
				bizVal := "project"
				bizIDVal := "111"

				header := make(http.Header)
				header.Set(bizKey, bizVal)
				header.Set(bizIDKey, bizIDVal)

				ctx.Request = httptest.NewRequest(http.MethodPost, "/product/sku/detail", nil)
				ctx.Request.Header = header
				require.Equal(t, bizVal, ctx.GetHeader(bizKey))
				require.Equal(t, bizIDVal, ctx.GetHeader(bizIDKey))
			},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {

				mockProvider := sessmocks.NewMockProvider(ctrl)
				mockSession := sessmocks.NewMockSession(ctrl)

				mockSession.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ekit.AnyValue{Err: fmt.Errorf("mock: 获取资源列表失败")})

				mockProvider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				return nil, mockProvider
			},
			wantCode: 403,
		},
		{
			name: "无X-Biz-ID请求头",
			before: func(t *testing.T, ctx *gin.Context) {
				t.Helper()

				bizKey := "X-Biz"
				bizIDKey := "X-Biz-IDs"
				bizVal := "project"
				bizIDVal := "112"

				header := make(http.Header)
				header.Set(bizKey, bizVal)
				header.Set(bizIDKey, bizIDVal)

				ctx.Request = httptest.NewRequest(http.MethodPost, "/product/sku/detail", nil)
				ctx.Request.Header = header
				require.Equal(t, bizVal, ctx.GetHeader(bizKey))
				require.Equal(t, bizIDVal, ctx.GetHeader(bizIDKey))
			},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {

				mockProvider := sessmocks.NewMockProvider(ctrl)
				mockSession := sessmocks.NewMockSession(ctrl)

				mockSession.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ekit.AnyValue{Val: ""})

				mockProvider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				return nil, mockProvider
			},
			wantCode: 403,
		},
		{
			name: "X-Biz-ID请求头其值非法",
			before: func(t *testing.T, ctx *gin.Context) {
				t.Helper()

				bizKey := "X-Biz"
				bizIDKey := "X-Biz-ID"
				bizVal := "project"
				bizIDVal := "invalidValue"

				header := make(http.Header)
				header.Set(bizKey, bizVal)
				header.Set(bizIDKey, bizIDVal)

				ctx.Request = httptest.NewRequest(http.MethodPost, "/product/sku/detail", nil)
				ctx.Request.Header = header
				require.Equal(t, bizVal, ctx.GetHeader(bizKey))
				require.Equal(t, bizIDVal, ctx.GetHeader(bizIDKey))
			},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {

				mockProvider := sessmocks.NewMockProvider(ctrl)
				mockSession := sessmocks.NewMockSession(ctrl)

				mockSession.EXPECT().Get(gomock.Any(), gomock.Any()).Return(ekit.AnyValue{Val: ""})

				mockProvider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				return nil, mockProvider
			},
			wantCode: 403,
		},
		{
			name: "实时查询资源权限出错",
			before: func(t *testing.T, ctx *gin.Context) {
				t.Helper()

				bizKey := "X-Biz"
				bizIDKey := "X-Biz-ID"
				bizVal := "project"
				bizIDVal := "107"

				header := make(http.Header)
				header.Set(bizKey, bizVal)
				header.Set(bizIDKey, bizIDVal)

				ctx.Request = httptest.NewRequest(http.MethodPost, "/product/sku/detail", nil)
				ctx.Request.Header = header
				require.Equal(t, bizVal, ctx.GetHeader(bizKey))
				require.Equal(t, bizIDVal, ctx.GetHeader(bizIDKey))
			},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {

				uid := int64(2666)
				biz := "project"
				bizId := int64(107)

				mockPermissionSvc := permissionmocks.NewMockService(ctrl)
				mockPermissionSvc.EXPECT().HasPermission(gomock.Any(), permission.PersonalPermission{
					Uid:   uid,
					Biz:   biz,
					BizID: bizId,
				}).Return(false, errors.New("mock: 查询权限出错"))

				mockProvider := sessmocks.NewMockProvider(ctrl)
				mockSession := sessmocks.NewMockSession(ctrl)

				mockSession.EXPECT().Get(gomock.Any(), biz).Return(ekit.AnyValue{Err: fmt.Errorf("%w", redis.Nil)})

				claims := session.Claims{
					Uid:  uid,
					SSID: "ssid-2666",
					Data: map[string]string{
						"memberDDL": strconv.FormatInt(time.Now().Add(1*24*time.Hour).UnixMilli(), 10),
					},
				}
				mockSession.EXPECT().Claims().Return(claims)

				mockProvider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				return mockPermissionSvc, mockProvider
			},
			wantCode: 403,
		},
		{
			name: "实时查询资源权限成功_用户无权限",
			before: func(t *testing.T, ctx *gin.Context) {
				t.Helper()

				bizKey := "X-Biz"
				bizIDKey := "X-Biz-ID"
				bizVal := "project"
				bizIDVal := "108"

				header := make(http.Header)
				header.Set(bizKey, bizVal)
				header.Set(bizIDKey, bizIDVal)

				ctx.Request = httptest.NewRequest(http.MethodPost, "/product/sku/detail", nil)
				ctx.Request.Header = header
				require.Equal(t, bizVal, ctx.GetHeader(bizKey))
				require.Equal(t, bizIDVal, ctx.GetHeader(bizIDKey))
			},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {

				uid := int64(2666)
				biz := "project"
				bizId := int64(108)

				mockPermissionSvc := permissionmocks.NewMockService(ctrl)

				mockPermissionSvc.EXPECT().HasPermission(gomock.Any(), permission.PersonalPermission{
					Uid:   uid,
					Biz:   biz,
					BizID: bizId,
				}).Return(false, nil)

				mockProvider := sessmocks.NewMockProvider(ctrl)
				mockSession := sessmocks.NewMockSession(ctrl)

				mockSession.EXPECT().Get(gomock.Any(), biz).Return(ekit.AnyValue{Err: fmt.Errorf("%w", redis.Nil)})

				claims := session.Claims{
					Uid:  uid,
					SSID: "ssid-2666",
					Data: map[string]string{
						"memberDDL": strconv.FormatInt(time.Now().Add(1*24*time.Hour).UnixMilli(), 10),
					},
				}
				mockSession.EXPECT().Claims().Return(claims)

				mockProvider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				return mockPermissionSvc, mockProvider
			},
			wantCode: 403,
		},
		{
			name: "设置session失败",
			before: func(t *testing.T, ctx *gin.Context) {
				t.Helper()

				bizKey := "X-Biz"
				bizIDKey := "X-Biz-ID"
				bizVal := "project"
				bizIDVal := "109"

				header := make(http.Header)
				header.Set(bizKey, bizVal)
				header.Set(bizIDKey, bizIDVal)

				ctx.Request = httptest.NewRequest(http.MethodPost, "/product/sku/detail", nil)
				ctx.Request.Header = header
				require.Equal(t, bizVal, ctx.GetHeader(bizKey))
				require.Equal(t, bizIDVal, ctx.GetHeader(bizIDKey))
			},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {

				uid := int64(2666)
				biz := "project"
				bizId := int64(109)

				mockPermissionSvc := permissionmocks.NewMockService(ctrl)

				mockPermissionSvc.EXPECT().HasPermission(gomock.Any(), permission.PersonalPermission{
					Uid:   uid,
					Biz:   biz,
					BizID: bizId,
				}).Return(true, nil)

				mockProvider := sessmocks.NewMockProvider(ctrl)
				mockSession := sessmocks.NewMockSession(ctrl)

				mockSession.EXPECT().Get(gomock.Any(), biz).Return(ekit.AnyValue{Err: fmt.Errorf("%w", redis.Nil)})

				claims := session.Claims{
					Uid:  uid,
					SSID: "ssid-2666",
					Data: map[string]string{
						"memberDDL": strconv.FormatInt(time.Now().Add(1*24*time.Hour).UnixMilli(), 10),
					},
				}
				mockSession.EXPECT().Claims().Return(claims)
				mockSession.EXPECT().Set(gomock.Any(), biz, "109").Return(errors.New("mock: 设置session出错"))

				mockProvider.EXPECT().Get(gomock.Any()).Return(mockSession, nil)

				return mockPermissionSvc, mockProvider
			},
			wantCode: 403,
		},
		{
			name:   "无JWT",
			before: func(t *testing.T, ctx *gin.Context) {},
			mock: func(ctrl *gomock.Controller) (permission.Service, session.Provider) {
				mockP := sessmocks.NewMockProvider(ctrl)
				mockP.EXPECT().Get(gomock.Any()).Return(nil, errors.New("mock no jwt"))
				return nil, mockP
			},
			wantCode: 403,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tc.before(t, c)

			svc, p := tc.mock(ctrl)
			builder := NewCheckPermissionMiddlewareBuilder(svc)
			builder.sp = p
			hdl := builder.Build()
			hdl(c)
			assert.Equal(t, tc.wantCode, c.Writer.Status())
		})
	}
}
