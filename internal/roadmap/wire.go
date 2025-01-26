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

package roadmap

import (
	"sync"

	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"github.com/ecodeclub/webook/internal/roadmap/internal/service/biz"

	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/roadmap/internal/repository"
	"github.com/ecodeclub/webook/internal/roadmap/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/roadmap/internal/service"
	"github.com/ecodeclub/webook/internal/roadmap/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

func InitModule(db *egorm.Component, queModule *baguwen.Module) *Module {
	wire.Build(
		web.NewAdminHandler,
		service.NewAdminService,
		NewConcurrentBizService,
		repository.NewCachedAdminRepository,
		initAdminDAO,

		web.NewHandler,
		service.NewService,
		repository.NewCachedRepository,
		dao.NewGORMRoadmapDAO,

		wire.Struct(new(Module), "*"),
		wire.FieldsOf(new(*baguwen.Module), "Svc", "SetSvc"),
	)
	return new(Module)
}

var (
	adminDAO    dao.AdminDAO
	daoInitOnce sync.Once
)

func initAdminDAO(db *egorm.Component) dao.AdminDAO {
	daoInitOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
		adminDAO = dao.NewGORMAdminDAO(db)
	})
	return adminDAO
}

func NewConcurrentBizService(questionSvc baguwen.Service, questionSetSvc baguwen.QuestionSetService) biz.Service {
	return biz.NewConcurrentBizService(map[string]biz.Strategy{
		domain.BizQuestion:    biz.NewQuestionStrategy(questionSvc),
		domain.BizQuestionSet: biz.NewQuestionSetStrategy(questionSetSvc),
	})
}
