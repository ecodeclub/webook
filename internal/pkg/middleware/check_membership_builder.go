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
	"time"

	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/gin-gonic/gin"
)

type CheckMembershipMiddlewareBuilder struct {
	svc    member.Service
	logger *elog.Component
	sp     session.Provider
}

func NewCheckMembershipMiddlewareBuilder(svc member.Service) *CheckMembershipMiddlewareBuilder {
	return &CheckMembershipMiddlewareBuilder{
		svc:    svc,
		logger: elog.DefaultLogger,
	}
}

func (c *CheckMembershipMiddlewareBuilder) Build() gin.HandlerFunc {
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

		claims := sess.Claims()
		memberDDL, _ := claims.Get("memberDDL").AsInt64()
		now := time.Now().UnixMilli()
		// 如果 jwt 中的数据格式不对，那么这里就会返回 0
		// jwt中找到会员截止日期，没有过期
		if memberDDL > now {
			return
		}

		elog.Debug("未开通过会员或会员已过期", elog.Int64("uid", claims.Uid),
			elog.String("ddl", time.UnixMilli(memberDDL).Format(time.DateTime)))

		// 1. jwt中未找到会员截止日期
		// 2. jwt中会员已经过期，有可能在这个期间，用户续费了会员，所以要再去实时查询一下
		// 查询svc
		info, err := c.svc.GetMembershipInfo(ctx, claims.Uid)
		if err != nil {
			elog.Error("查询会员失败", elog.Int64("uid", claims.Uid), elog.FieldErr(err))
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}

		if info.EndAt == 0 {
			elog.Debug("未开通会员", elog.Int64("uid", claims.Uid),
				elog.String("ddl", time.UnixMilli(info.EndAt).Format(time.DateTime)))
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}

		if info.EndAt < now {
			elog.Debug("会员已过期", elog.Int64("uid", claims.Uid),
				elog.String("ddl", time.UnixMilli(info.EndAt).Format(time.DateTime)))
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}

		// 在原有jwt数据中添加会员截止日期
		jwtData := claims.Data
		jwtData["memberDDL"] = strconv.FormatInt(info.EndAt, 10)
		claims.Data = jwtData
		err = c.sp.UpdateClaims(gctx, claims)
		if err != nil {
			elog.Error("重新生成 token 失败", elog.Int64("uid", claims.Uid), elog.FieldErr(err))
			gctx.AbortWithStatus(http.StatusForbidden)
			return
		}
	}
}
