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

//go:build wireinject

package user

import (
	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/user/internal/repository"
	"github.com/ecodeclub/webook/internal/user/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	iniHandler,
	cache.NewUserECache,
	initDAO,
	initWechatWebOAuthService,
	initWechatMiniOAuthService,
	initRegistrationEventProducer,
	service.NewUserService,
	repository.NewCachedUserRepository)

func InitModule(db *egorm.Component,
	cache ecache.Cache,
	q mq.MQ, creators []string,
	memberSvc *member.Module,
	sp session.Provider,
	permissionSvc *permission.Module) *Module {
	wire.Build(
		ProviderSet,
		wire.FieldsOf(new(*member.Module), "Svc"),
		wire.FieldsOf(new(*permission.Module), "Svc"),
		wire.Struct(new(Module), "*"),
	)
	return new(Module)
}
