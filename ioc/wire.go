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

package ioc

import (
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/cos"
	"github.com/ecodeclub/webook/internal/label"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/pkg/middleware"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/skill"
	"github.com/google/wire"
)

var BaseSet = wire.NewSet(InitDB, InitCache, InitRedis, InitMQ, InitCosConfig)

func InitApp() (*App, error) {
	wire.Build(wire.Struct(new(App), "*"),
		BaseSet,
		cos.InitHandler,
		baguwen.InitModule,
		wire.FieldsOf(new(*baguwen.Module), "Hdl", "QsHdl"),
		InitUserHandler,
		InitSession,
		label.InitHandler,
		cases.InitModule,
		wire.FieldsOf(new(*cases.Module), "Hdl"),
		skill.InitHandler,
		// 会员服务
		member.InitModule,
		wire.FieldsOf(new(*member.Module), "Svc"),
		// 会员检查中间件
		middleware.NewCheckMembershipMiddlewareBuilder,
		initGinxServer)
	return new(App), nil
}
