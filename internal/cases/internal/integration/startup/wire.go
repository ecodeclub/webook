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
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/cases/internal/event"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
	"github.com/ecodeclub/webook/internal/interactive"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitModule(
	syncProducer event.SyncEventProducer,
	aiModule *ai.Module,
	intrModule *interactive.Module) (*cases.Module, error) {
	wire.Build(cases.InitCaseDAO,
		testioc.BaseSet,
		dao.NewCaseSetDAO,
		dao.NewGORMExamineDAO,
		repository.NewCaseRepo,
		repository.NewCaseSetRepo,
		repository.NewCachedExamineRepository,
		event.NewInteractiveEventProducer,
		service.NewService,
		service.NewCaseSetService,
		service.NewLLMExamineService,
		web.NewHandler,
		web.NewAdminCaseSetHandler,
		web.NewAdminCaseHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*ai.Module), "Svc"),
		wire.Struct(new(cases.Module), "AdminHandler", "ExamineSvc", "Svc", "Hdl", "AdminSetHandler"),
	)
	return new(cases.Module), nil
}

func InitExamModule(
	syncProducer event.SyncEventProducer,
	intrModule *interactive.Module,
	aiModule *ai.Module) (*cases.Module, error) {
	wire.Build(
		testioc.BaseSet,
		cases.InitCaseDAO,
		dao.NewCaseSetDAO,
		dao.NewGORMExamineDAO,
		repository.NewCaseRepo,
		repository.NewCaseSetRepo,
		repository.NewCachedExamineRepository,
		event.NewInteractiveEventProducer,
		service.NewCaseSetService,
		service.NewService,
		service.NewLLMExamineService,
		web.NewHandler,
		web.NewAdminCaseSetHandler,
		web.NewAdminCaseHandler,
		web.NewExamineHandler,
		web.NewCaseSetHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*ai.Module), "Svc"),
		wire.Struct(new(cases.Module), "*"),
	)
	return new(cases.Module), nil
}
