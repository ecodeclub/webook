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
	"context"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/user/internal/event"
	"github.com/ecodeclub/webook/internal/user/internal/repository"
	"github.com/ecodeclub/webook/internal/user/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/user/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/ecodeclub/webook/internal/user/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"github.com/gotomicro/ego/core/econf"
)

var ProviderSet = wire.NewSet(web.NewHandler,
	cache.NewUserECache,
	InitDAO,
	InitWechatService,
	InitRegistrationEventProducer,
	service.NewUserService,
	repository.NewCachedUserRepository)

func InitHandler(db *egorm.Component, cache ecache.Cache, q mq.MQ, creators []string, memberSvc member.Service) *Handler {
	wire.Build(ProviderSet)
	return new(Handler)
}

func InitWechatService() service.OAuth2Service {
	type Config struct {
		AppSecretID      string `yaml:"appSecretID"`
		AppSecretKey     string `yaml:"appSecretKey"`
		LoginRedirectURL string `yaml:"loginRedirectURL"`
	}
	var cfg Config
	err := econf.UnmarshalKey("wechat", &cfg)
	if err != nil {
		panic(err)
	}
	return service.NewWechatService(cfg.AppSecretID, cfg.AppSecretKey, cfg.LoginRedirectURL)
}

func InitDAO(db *egorm.Component) dao.UserDAO {
	err := dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return dao.NewGORMUserDAO(db)
}

func InitRegistrationEventProducer(q mq.MQ) *event.RegistrationEventProducer {
	type Config struct {
		Topic      string `yaml:"topic"`
		Partitions int    `yaml:"partitions"`
	}
	var cfg Config
	err := econf.UnmarshalKey("user.event", &cfg)
	if err != nil {
		panic(err)
	}
	err = q.CreateTopic(context.Background(), cfg.Topic, cfg.Partitions)
	if err != nil {
		panic(err)
	}
	producer, err := q.Producer(cfg.Topic)
	if err != nil {
		panic(err)
	}
	return event.NewRegistrationEventProducer(producer)
}

// Handler 暴露出去给 ioc 使用
type Handler = web.Handler
