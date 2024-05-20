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
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"github.com/redis/go-redis/v9"
)

type CheckPermissionMiddlewareBuilder struct {
	svc    permission.Service
	logger *elog.Component
	sp     session.Provider
}

func NewCheckPermissionMiddlewareBuilder(svc permission.Service) *CheckPermissionMiddlewareBuilder {
	return &CheckPermissionMiddlewareBuilder{
		svc:    svc,
		logger: elog.DefaultLogger,
	}
}

func (c *CheckPermissionMiddlewareBuilder) Build() gin.HandlerFunc {
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

		// 获取资源列表
		resourceName := ctx.GetHeader("X-Biz")
		if len(resourceName) == 0 {
			gctx.AbortWithStatus(http.StatusForbidden)
			c.logger.Debug("biz非法", elog.FieldErr(err))
			return
		}
		resourceStr, err := sess.Get(ctx.Request.Context(), resourceName).AsString()
		if err != nil && !errors.Is(err, redis.Nil) {
			gctx.AbortWithStatus(http.StatusForbidden)
			c.logger.Debug("biz非法", elog.FieldErr(err))
			return
		}
		var resources []string
		if len(resourceStr) > 0 {
			resources = strings.Split(resourceStr, ",")
		}

		// 获取资源Id
		resourceIdStr := ctx.GetHeader("X-Biz-ID")
		resourceId, err := strconv.ParseInt(resourceIdStr, 10, 64)
		if len(resourceIdStr) == 0 || err != nil {
			gctx.AbortWithStatus(http.StatusForbidden)
			c.logger.Debug("bizId非法", elog.FieldErr(errors.New("无法获取bizId")))
			return
		}

		// resourceId 是否在 resources 中, 快路径
		log.Printf("resouces = %#v\n", resources)
		_, ok := slice.Find(resources, func(src string) bool {
			return src == resourceIdStr
		})
		if ok {
			return
		}

		// 如果不在,实时查询permission模块,并将验证通过的 resource_id 放入 resources 中
		uid := sess.Claims().Uid
		ok, err = c.svc.HasPermission(ctx.Request.Context(), permission.PersonalPermission{
			Uid:   uid,
			Biz:   resourceName,
			BizID: resourceId,
		})
		if err != nil || !ok {
			gctx.AbortWithStatus(http.StatusForbidden)
			c.logger.Debug("用户无权限", elog.FieldErr(err))
			return
		}
		resources = append(resources, resourceIdStr)

		// 更新Session
		err = sess.Set(ctx.Request.Context(), resourceName, strings.Join(resources, ","))
		if err != nil {
			elog.Error("更新Session失败", elog.Int64("uid", uid), elog.FieldErr(err))
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}
	}
}
