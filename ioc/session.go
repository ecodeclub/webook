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

package ioc

import (
	"time"

	"github.com/ecodeclub/ginx/session/header"
	"github.com/ecodeclub/ginx/session/mixin"

	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/ginx/session/cookie"
	redis2 "github.com/ecodeclub/ginx/session/redis"
	"github.com/gotomicro/ego/core/econf"
	"github.com/redis/go-redis/v9"
)

func InitSession(cmd redis.Cmdable) session.Provider {
	type Config struct {
		SessionEncryptedKey string `yaml:"sessionEncryptedKey"`
		Cookie              struct {
			Domain string `yaml:"domain"`
		} `yaml:"cookie"`
	}
	var cfg Config
	err := econf.UnmarshalKey("session", &cfg)
	if err != nil {
		panic(err)
	}
	// 默认是一天
	const day = time.Hour * 24
	sp := redis2.NewSessionProvider(cmd, cfg.SessionEncryptedKey, day)
	cookieC := &cookie.TokenCarrier{
		MaxAge:   int(day.Seconds()),
		Name:     "ssid",
		Secure:   true,
		HttpOnly: true,
		Domain:   cfg.Cookie.Domain,
	}
	headerC := header.NewTokenCarrier()
	sp.TokenCarrier = mixin.NewTokenCarrier(headerC, cookieC)
	return sp
}
