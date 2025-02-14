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
	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/ego-component/egorm"
	"github.com/gotomicro/ego/core/econf"
)

func InitUserModule(db *egorm.Component,
	sp session.Provider,
	ec ecache.Cache,
	q mq.MQ,
	memModule *member.Module,
	perm *permission.Module) *user.Module {
	type UserConfig struct {
		Creators []string `json:"creators"`
	}
	var cfg UserConfig
	err := econf.UnmarshalKey("user", &cfg)
	if err != nil {
		panic(err)
	}
	return user.InitModule(db, ec, q, cfg.Creators, memModule, sp, perm)
}
