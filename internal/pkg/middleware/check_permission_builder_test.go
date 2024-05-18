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
	"net/http/httptest"
	"testing"

	"github.com/ecodeclub/webook/internal/permission"
	"github.com/stretchr/testify/assert"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/gin-gonic/gin"
	"go.uber.org/mock/gomock"
)

func TestCheckPermission(t *testing.T) {
	testCases := []struct {
		name      string
		mock      func(ctrl *gomock.Controller) (permission.Service, session.Provider)
		req       any
		wantCode  int
		afterFunc func(t *testing.T, ctx *ginx.Context)
	}{
		// JWT中有资源列表,资源在列表中,检查通过
		// JWT中有资源列表,资源不在列表中,检查通过,更新session成功
		// JWT中无资源列表,实时查询,用户有权限,更新session成功
		// 实时查询资源权限出错
		// 实时查询资源权限成功,但用户无权限
		// 无JWT
		{},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			svc, p := tc.mock(ctrl)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			builder := NewCheckPermissionMiddlewareBuilder(svc, tc.req)
			builder.sp = p
			hdl := builder.Build("project")
			hdl(c)
			assert.Equal(t, tc.wantCode, c.Writer.Status())
		})
	}
}
