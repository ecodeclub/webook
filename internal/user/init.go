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
	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/user/internal/event"
	"github.com/ecodeclub/webook/internal/user/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/ecodeclub/webook/internal/user/internal/web"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
)

func iniHandler(
	weSvc wechatWebOAuth2Service,
	weMiniSvc wechatMiniOAuth2Service,
	userSvc service.UserService,
	memberSvc member.Service,
	sp session.Provider,
	permissionSvc permission.Service, creators []string) *Handler {
	return web.NewHandler(weSvc, weMiniSvc, userSvc, memberSvc, permissionSvc, sp, creators)
}

func initWechatMiniOAuthService() wechatMiniOAuth2Service {
	type Config struct {
		AppSecretID  string `yaml:"appSecretID"`
		AppSecretKey string `yaml:"appSecretKey"`
	}
	var cfg Config
	err := econf.UnmarshalKey("wechat.mini", &cfg)
	if err != nil {
		panic(err)
	}
	return service.NewWechatMiniService(cfg.AppSecretID, cfg.AppSecretKey)
}

func initWechatWebOAuthService(cache ecache.Cache) wechatWebOAuth2Service {
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
	return service.NewWechatService(cache, cfg.AppSecretID, cfg.AppSecretKey, cfg.LoginRedirectURL)
}

func initDAO(db *egorm.Component) dao.UserDAO {
	err := dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return dao.NewGORMUserDAO(db)
}

func initRegistrationEventProducer(q mq.MQ) event.RegistrationEventProducer {
	producer, err := event.NewRegistrationEventProducer(q)
	if err != nil {
		panic(err)
	}
	return producer
}
