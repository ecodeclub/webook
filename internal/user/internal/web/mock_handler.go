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

//go:build mock

package web

import (
	"strconv"
	"time"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
)

// MockLogin 模拟的，用来开发测试环境省略登录过程
func (h *Handler) MockLogin(ctx *ginx.Context) (ginx.Result, error) {
	const uid = 1
	profile, err := h.userSvc.Profile(ctx, uid)
	if err != nil {
		return systemErrorResult, err
	}
	// 构建session
	jwtData := map[string]string{}
	jwtData["creator"] = strconv.FormatBool(true)
	// 设置会员截止日期
	memberDDL := time.Now().Add(time.Hour * 24).UnixMilli()
	jwtData["memberDDL"] = strconv.FormatInt(memberDDL, 10)

	_, err = session.NewSessionBuilder(ctx, uid).SetJwtData(jwtData).Build()
	if err != nil {
		return systemErrorResult, err
	}
	res := newProfile(profile)
	res.IsCreator = true
	res.MemberDDL = memberDDL
	return ginx.Result{
		Msg:  "OK",
		Data: res,
	}, nil
}
