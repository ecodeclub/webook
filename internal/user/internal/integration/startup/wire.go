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

package startup

import (
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/permission"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/ecodeclub/webook/internal/user/internal/event"
	"github.com/ecodeclub/webook/internal/user/internal/repository"
	"github.com/ecodeclub/webook/internal/user/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/user/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/ecodeclub/webook/internal/user/internal/web"
	"github.com/google/wire"
)

func InitHandler(weSvc wechatWebOAuth2Service,
	weMiniSvc wechatMiniOAuth2Service,
	mem *member.Module,
	perm *permission.Module,
	sp session.Provider,
	creators []string) *user.Handler {
	wire.Build(iniHandler,
		testioc.BaseSet,
		wire.FieldsOf(new(*member.Module), "Svc"),
		wire.FieldsOf(new(*permission.Module), "Svc"),
		initRegistrationEventProducer,
		service.NewUserService,
		dao.NewGORMUserDAO,
		cache.NewUserECache,
		repository.NewCachedUserRepository)
	return new(user.Handler)
}

func iniHandler(
	weSvc wechatWebOAuth2Service,
	weMiniSvc wechatMiniOAuth2Service,
	userSvc service.UserService,
	memberSvc member.Service,
	permissionSvc permission.Service,
	sp session.Provider,
	creators []string) *web.Handler {
	return web.NewHandler(weSvc, weMiniSvc, userSvc, memberSvc, permissionSvc, sp, creators)
}
func InitModule() *user.Module {
	wire.Build(
		testioc.BaseSet,
		initRegistrationEventProducer,
		service.NewUserService,
		dao.NewGORMUserDAO,
		cache.NewUserECache,
		repository.NewCachedUserRepository,
		wire.Struct(new(user.Module), "Svc"),
	)
	return new(user.Module)
}

func initRegistrationEventProducer(q mq.MQ) event.RegistrationEventProducer {
	p, err := event.NewRegistrationEventProducer(q)
	if err != nil {
		panic(err)
	}
	return p
}
