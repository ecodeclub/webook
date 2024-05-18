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
	"strconv"
	"strings"

	"github.com/ecodeclub/ekit/slice"
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

func NewCheckPermissionMiddlewareBuilder[Req any](svc permission.Service, req Req) *CheckPermissionMiddlewareBuilder[Req] {
	return &CheckPermissionMiddlewareBuilder[Req]{
		svc:    svc,
		req:    req,
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

		// 1. 从session中获取 resources
		claims := sess.Claims()
		resourceStr, _ := claims.Get(biz).AsString() // map["project"]"101,103,102"
		resources := strings.Split(resourceStr, ",") // [101,103,102]

		// 2. 获取当前请求者uid
		// TODO: 如何在中间件中获取Uid
		uid := int64(0)

		// 3. 获取当前待访问资源的id
		// TODO: 如何获取待访问资源的id, 以project为例, project_id,请求通常为POST id通常在body中
		// 先读出来?在放进去? 这也是引入范型参数req的原因
		rid := int64(0)
		// err = ctx.Bind(c.req)
		// if err != nil {
		// }

		// 4. resource_id 是否在 resources 中, 快路径
		_, ok := slice.Find(resources, func(src string) bool {
			i, _ := strconv.ParseInt(src, 10, 64)
			return i == rid
		})
		if ok {
			return
		}

		// 5. 如果不在,实时查询permission模块,并将验证通过的 resource_id 放入 resources 中
		ok, err = c.svc.HasPersonalPermission(ctx.Request.Context(), permission.PersonalPermission{
			Uid:   uid,
			Biz:   biz,
			BizID: rid,
		})
		if err != nil || !ok {
			gctx.AbortWithStatus(http.StatusForbidden)
			c.logger.Debug("用户无权限", elog.FieldErr(err))
			return
		}
		resources = append(resources, strconv.FormatInt(rid, 10))

		// 6.更新Session
		jwtData := claims.Data
		jwtData[biz] = strings.Join(resources, ",")
		claims.Data = jwtData
		err = c.sp.UpdateClaims(gctx, claims)
		if err != nil {
			elog.Error("重新生成 token 失败", elog.Int64("uid", claims.Uid), elog.FieldErr(err))
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}
	}
}
