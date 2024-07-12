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
	"github.com/ecodeclub/webook/internal/interactive"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/question/internal/event"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"github.com/ecodeclub/webook/internal/question/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/ecodeclub/webook/internal/question/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitModule(p event.SyncDataToSearchEventProducer,
	intrModule *interactive.Module) (*baguwen.Module, error) {
	wire.Build(
		testioc.BaseSet,
		moduleSet,
		event.NewInteractiveEventProducer,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
	)
	return new(baguwen.Module), nil
}

var moduleSet = wire.NewSet(baguwen.InitQuestionDAO,
	cache.NewQuestionECache,
	repository.NewCacheRepository,
	service.NewService,
	web.NewHandler,
	web.NewAdminHandler,
	baguwen.ExamineHandlerSet,
	baguwen.InitQuestionSetDAO,
	repository.NewQuestionSetRepository,
	service.NewQuestionSetService,
	web.NewQuestionSetHandler,
	wire.Struct(new(baguwen.Module), "*"),
)
