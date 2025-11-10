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

package skill

import (
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/skill/internal/event"

	"github.com/ecodeclub/webook/internal/cases"
	baguwen "github.com/ecodeclub/webook/internal/question"

	"github.com/ecodeclub/webook/internal/skill/internal/repository"
	"github.com/ecodeclub/webook/internal/skill/internal/repository/cache"
	dao2 "github.com/ecodeclub/webook/internal/skill/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/skill/internal/service"
	"github.com/ecodeclub/webook/internal/skill/internal/web"

	"github.com/ecodeclub/ecache"

	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitHandler(
	db *egorm.Component,
	ec ecache.Cache,
	queModule *baguwen.Module,
	caseModule *cases.Module,
	q mq.MQ) (*Handler, error) {
	wire.Build(
		InitSkillDAO,
		wire.FieldsOf(new(*baguwen.Module), "Svc", "SetSvc"),
		wire.FieldsOf(new(*cases.Module), "ExamineSvc", "Svc", "SetSvc"),
		cache.NewSkillCache,
		repository.NewSkillRepo,
		event.NewSyncEventProducer,
		service.NewSkillService,
		web.NewHandler,
	)
	return new(Handler), nil
}

var daoOnce = sync.Once{}

func InitTableOnce(db *gorm.DB) {
	daoOnce.Do(func() {
		err := dao2.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
}

func InitSkillDAO(db *egorm.Component) dao2.SkillDAO {
	InitTableOnce(db)
	return dao2.NewSkillDAO(db)
}

type Handler = web.Handler
