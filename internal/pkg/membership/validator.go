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
	"net/http"
	"time"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/member"
)

var (
	ErrMembershipExpired = errors.New("会员已过期")
	ErrGetMemberInfo     = errors.New("获取会员信息失败")
	ErrSessionGeneration = errors.New("生成新session失败")
)

type Validator struct {
	svc member.Service
}

func NewValidator(svc member.Service) *Validator {
	return &Validator{
		svc: svc,
	}
}

func (c *Validator) Membership(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	claims := sess.Claims()
	memberDDL, err := time.Parse(time.DateTime, claims.Get("memberDDL").StringOrDefault(""))
	if err == nil {
		// jwt中找到会员截止日期
		if memberDDL.Local().Compare(time.Now().Local()) <= 0 {
			ctx.AbortWithStatus(http.StatusUnauthorized)
			// todo: 替换为 ginx.ErrUnauthorized
			return ginx.Result{}, fmt.Errorf("%w uid: %d", ErrMembershipExpired, claims.Uid)
		}
		return ginx.Result{}, nil
	}

	// jwt中未找到会员截止日期
	// 查询svc
	info, err := c.svc.GetMembershipInfo(ctx, claims.Uid)
	if err != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		// todo: 替换为 ginx.ErrUnauthorized
		return ginx.Result{}, fmt.Errorf("%w uid: %d", ErrGetMemberInfo, claims.Uid)
	}

	if time.Unix(info.StartAt, 0).Local().Compare(time.Now().Local()) <= 0 {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		// todo: 替换为 ginx.ErrUnauthorized
		return ginx.Result{}, fmt.Errorf("%w uid: %d", ErrMembershipExpired, claims.Uid)
	}

	// 再原有jwt数据中添加会员截止日期
	jwtData := claims.Data
	jwtData["memberDDL"] = time.Unix(info.EndAt, 0).Local().Format(time.DateTime)

	// 刷新session
	if session.RenewAccessToken(ctx) != nil {
		ctx.AbortWithStatus(http.StatusUnauthorized)
		// todo: 替换为 ginx.ErrUnauthorized
		return ginx.Result{}, fmt.Errorf("%w uid: %d", ErrSessionGeneration, claims.Uid)
	}

	return ginx.Result{}, nil
}
