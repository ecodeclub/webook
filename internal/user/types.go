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

package user

import (
	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/ecodeclub/webook/internal/user/internal/web"
)

// Handler 暴露出去给 ioc 使用
type Handler = web.Handler
type User = domain.User
type WechatInfo = domain.WechatInfo

// UserService 方便测试
type UserService = service.UserService

type Module struct {
	Hdl *Handler
	Svc UserService
}

// 规避 wire 的坑
type wechatMiniOAuth2Service service.OAuth2Service
type wechatWebOAuth2Service service.OAuth2Service
