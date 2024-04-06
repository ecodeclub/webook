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

package membership

import (
	"errors"
	"fmt"
	"time"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/gin-gonic/gin"
)

var (
	ErrMembershipExpired      = errors.New("会员已过期")
	ErrGetMemberInfo          = errors.New("获取会员信息失败")
	ErrRenewAccessTokenFailed = errors.New("刷新AccessToken失败")
)

type CheckMembershipMiddlewareBuilder struct {
	svc member.Service
}

func NewCheckMembershipMiddlewareBuilder(svc member.Service) *CheckMembershipMiddlewareBuilder {
	return &CheckMembershipMiddlewareBuilder{
		svc: svc,
	}
}

func (c *CheckMembershipMiddlewareBuilder) check(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	claims := sess.Claims()
	memberDDL, err := time.Parse(time.DateTime, claims.Get("memberDDL").StringOrDefault(""))
	if err == nil {
		// jwt中找到会员截止日期
		if memberDDL.UTC().Compare(time.Now().UTC()) <= 0 {
			return ginx.Result{}, fmt.Errorf("%w: %w: uid: %d", ginx.ErrUnauthorized, ErrMembershipExpired, claims.Uid)
		}
		return ginx.Result{}, nil
	}

	// jwt中未找到会员截止日期
	// 查询svc
	info, err := c.svc.GetMembershipInfo(ctx, claims.Uid)
	if err != nil {
		return ginx.Result{}, fmt.Errorf("%w: %w: uid: %d", ginx.ErrUnauthorized, ErrGetMemberInfo, claims.Uid)
	}

	memberDDL = time.UnixMilli(info.EndAt).UTC()
	if memberDDL.Compare(time.Now().UTC()) <= 0 {
		return ginx.Result{}, fmt.Errorf("%w: %w: uid: %d", ginx.ErrUnauthorized, ErrMembershipExpired, claims.Uid)
	}

	// 再原有jwt数据中添加会员截止日期
	jwtData := claims.Data
	jwtData["memberDDL"] = memberDDL.Format(time.DateTime)

	if session.RenewAccessToken(ctx) != nil {
		return ginx.Result{}, fmt.Errorf("%w: %w: uid: %d", ginx.ErrUnauthorized, ErrRenewAccessTokenFailed, claims.Uid)
	}

	return ginx.Result{}, nil
}

func (c *CheckMembershipMiddlewareBuilder) Build() gin.HandlerFunc {
	return ginx.S(c.check)
}
