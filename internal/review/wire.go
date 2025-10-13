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

package review

import (
	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/company"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/review/internal/event"
	"github.com/ecodeclub/webook/internal/review/internal/repository"
	"github.com/ecodeclub/webook/internal/review/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/review/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/review/internal/service"
	"github.com/ecodeclub/webook/internal/review/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

func InitModule(db *egorm.Component,
	interSvc *interactive.Module,
	companyModule *company.Module,
	q mq.MQ,
	sp session.Provider,
	ec ecache.Cache,
) *Module {
	wire.Build(
		initReviewDao,
		initIntrProducer,
		repository.NewReviewRepo,
		service.NewReviewSvc,
		cache.NewReviewCache,
		web.NewHandler,
		web.NewAdminHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*company.Module), "Svc"),
		wire.Struct(new(Module), "*"),
	)
	return new(Module)
}

func initReviewDao(db *egorm.Component) dao.ReviewDAO {
	err := dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return dao.NewReviewDAO(db)
}

func initIntrProducer(q mq.MQ) event.InteractiveEventProducer {
	producer, err := event.NewInteractiveEventProducer(q)
	if err != nil {
		panic(err)
	}
	return producer
}
