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
	"net/http"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type CheckPermissionMiddlewareBuilder[Req any] struct {
	svc    permission.Service
	req    Req
	logger *elog.Component
	sp     session.Provider
}

func NewCheckPermissionMiddlewareBuilder[Req any](svc permission.Service) *CheckPermissionMiddlewareBuilder[Req] {
	return &CheckPermissionMiddlewareBuilder[Req]{
		svc:    svc,
		logger: elog.DefaultLogger,
	}
}

func (c *CheckPermissionMiddlewareBuilder[Req]) Build(biz string) gin.HandlerFunc {
	if c.sp == nil {
		c.sp = session.DefaultProvider()
	}
	return func(ctx *gin.Context) {
		gctx := &ginx.Context{Context: ctx}
		sess, err := c.sp.Get(gctx)
		if err != nil {
			gctx.AbortWithStatus(http.StatusForbidden)
			c.logger.Debug("用户未登录", elog.FieldErr(err))
			return
		}

		// 1. 从session中获取resourceSet
		// 2. 获取当前请求者uid
		// 3. 获取当前资源的id, resource_id
		// 4. resource_id是否在resourceSet中, 快路径
		// 5. 如果不在,实时查询permission模块
		ok, err := c.svc.HasPersonalPermission(ctx.Request.Context(), permission.PersonalPermission{
			Uid:   0,   // 当前 请求者的UID
			Biz:   biz, // project
			BizID: 0,   // 当前 访问资源的ID, project_id
		})
		if err != nil || !ok {
			gctx.AbortWithStatus(http.StatusForbidden)
			c.logger.Debug("用户无权限", elog.FieldErr(err))
			return
		}
		// 6.验证通过, 将 resource_id 放入 resourceSet中,然后更新Session
		// TODO: 如何在中间件中获取Uid
		// TODO: 如何获取待访问资源的BizId, 以project为例, project_id,
		//
		claims := sess.Claims()

		// 在原有jwt数据中添加会员截止日期
		jwtData := claims.Data
		jwtData[biz] = "1,2,3,3,4"
		claims.Data = jwtData
		err = c.sp.UpdateClaims(gctx, claims)
		if err != nil {
			elog.Error("重新生成 token 失败", elog.Int64("uid", claims.Uid), elog.FieldErr(err))
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}
	}
}
