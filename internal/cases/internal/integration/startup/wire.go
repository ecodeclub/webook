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
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/cases/internal/event"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/member"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitModule(
	syncProducer event.SyncEventProducer,
	knowledgeBaseProducer event.KnowledgeBaseEventProducer,
	aiModule *ai.Module,
	memberModule *member.Module,
	sp session.Provider,
	intrModule *interactive.Module) (*cases.Module, error) {
	wire.Build(cases.InitCaseDAO,
		testioc.BaseSet,
		dao.NewCaseSetDAO,
		dao.NewGORMExamineDAO,
		cache.NewCaseCache,
		repository.NewCaseRepo,
		repository.NewCaseSetRepo,
		repository.NewCachedExamineRepository,
		event.NewInteractiveEventProducer,
		service.NewService,
		service.NewCaseSetService,
		service.NewLLMExamineService,
		initKnowledgeBaseSvc,
		web.NewHandler,
		web.NewAdminCaseSetHandler,
		web.NewAdminCaseHandler,
		web.NewKnowledgeBaseHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*member.Module), "Svc"),
		wire.FieldsOf(new(*ai.Module), "Svc", "KnowledgeBaseSvc"),
		wire.Struct(new(cases.Module), "AdminHandler", "ExamineSvc", "Svc", "Hdl", "AdminSetHandler", "KnowledgeBaseHandler"),
	)
	return new(cases.Module), nil
}

func InitExamModule(
	syncProducer event.SyncEventProducer,
	knowledgeBaseProducer event.KnowledgeBaseEventProducer,
	intrModule *interactive.Module,
	memberModule *member.Module,
	sp session.Provider,
	aiModule *ai.Module) (*cases.Module, error) {
	wire.Build(
		testioc.BaseSet,
		cases.InitCaseDAO,
		dao.NewCaseSetDAO,
		dao.NewGORMExamineDAO,
		cache.NewCaseCache,
		repository.NewCaseRepo,
		repository.NewCaseSetRepo,
		repository.NewCachedExamineRepository,
		event.NewInteractiveEventProducer,
		service.NewCaseSetService,
		service.NewService,
		service.NewLLMExamineService,
		initKnowledgeBaseSvc,
		web.NewHandler,
		web.NewAdminCaseSetHandler,
		web.NewAdminCaseHandler,
		web.NewExamineHandler,
		web.NewCaseSetHandler,
		web.NewKnowledgeBaseHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*ai.Module), "Svc", "KnowledgeBaseSvc"),
		wire.Struct(new(cases.Module), "*"),
		wire.FieldsOf(new(*member.Module), "Svc"),
	)
	return new(cases.Module), nil
}

func initKnowledgeBaseSvc(svc ai.KnowledgeBaseService, caRepo repository.CaseRepo) service.KnowledgeBaseService {
	return service.NewKnowledgeBaseService(caRepo, svc, "knowledge_id")
}
