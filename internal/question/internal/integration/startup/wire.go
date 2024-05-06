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
	"github.com/ecodeclub/ecache"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/question/internal/event"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"github.com/ecodeclub/webook/internal/question/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/ecodeclub/webook/internal/question/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

func InitHandler(p event.SyncEventProducer) (*web.Handler, error) {
	wire.Build(
		testioc.BaseSet,
		initModule,
		wire.FieldsOf(new(*baguwen.Module), "Hdl"),
	)
	return new(web.Handler), nil
}

func initModule(db *egorm.Component, ec ecache.Cache, p event.SyncEventProducer) (*baguwen.Module, error) {
	wire.Build(baguwen.InitQuestionDAO,
		cache.NewQuestionECache,
		repository.NewCacheRepository,
		service.NewService,
		web.NewHandler,
		baguwen.InitQuestionSetDAO,
		repository.NewQuestionSetRepository,
		service.NewQuestionSetService,
		web.NewQuestionSetHandler,
		wire.Struct(new(baguwen.Module), "*"),
	)
	return new(baguwen.Module), nil
}

func InitQuestionSetHandler(p event.SyncEventProducer) (*web.QuestionSetHandler, error) {
	wire.Build(testioc.BaseSet, initModule,
		wire.FieldsOf(new(*baguwen.Module), "QsHdl"))
	return new(web.QuestionSetHandler), nil
}
