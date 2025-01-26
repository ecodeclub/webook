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

package baguwen

import (
	"sync"

	"github.com/ecodeclub/ginx/session"

	"github.com/ecodeclub/webook/internal/member"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/gotomicro/ego/core/econf"

	"github.com/ecodeclub/webook/internal/question/internal/job"

	"github.com/ecodeclub/webook/internal/permission"

	"github.com/ecodeclub/webook/internal/interactive"

	"github.com/ecodeclub/webook/internal/question/internal/event"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/mq-api"

	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"github.com/ecodeclub/webook/internal/question/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/ecodeclub/webook/internal/question/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

var ExamineHandlerSet = wire.NewSet(
	web.NewExamineHandler,
	service.NewLLMExamineService,
	repository.NewCachedExamineRepository,
	dao.NewGORMExamineDAO)

func InitModule(db *egorm.Component,
	intrModule *interactive.Module,
	ec ecache.Cache,
	perm *permission.Module,
	aiModule *ai.Module,
	memberModule *member.Module,
	sp session.Provider,
	q mq.MQ) (*Module, error) {
	wire.Build(InitQuestionDAO,
		cache.NewQuestionECache,
		repository.NewCacheRepository,
		event.NewSyncEventProducer,
		event.NewInteractiveEventProducer,
		InitKnowledgeBaseUploadProducer,
		service.NewService,
		web.NewHandler,
		web.NewAdminHandler,
		web.NewAdminQuestionSetHandler,

		ExamineHandlerSet,

		InitQuestionSetDAO,
		repository.NewQuestionSetRepository,
		service.NewQuestionSetService,
		web.NewQuestionSetHandler,
		initKnowledgeStarter,
		InitKnowledgeBaseSvc,
		web.NewKnowledgeBaseHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*permission.Module), "Svc"),
		wire.FieldsOf(new(*member.Module), "Svc"),

		wire.FieldsOf(new(*ai.Module), "Svc", "KnowledgeBaseSvc"),

		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
}

var daoOnce = sync.Once{}

func initKnowledgeStarter(svc service.Service) *job.KnowledgeJobStarter {
	baseDir := econf.GetString("job.genKnowledge.baseDir")
	return job.NewKnowledgeJobStarter(svc, baseDir)
}

func InitTableOnce(db *gorm.DB) {
	daoOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
}

func InitQuestionDAO(db *egorm.Component) dao.QuestionDAO {
	InitTableOnce(db)
	return dao.NewGORMQuestionDAO(db)
}

func InitQuestionSetDAO(db *egorm.Component) dao.QuestionSetDAO {
	InitTableOnce(db)
	return dao.NewGORMQuestionSetDAO(db)
}
