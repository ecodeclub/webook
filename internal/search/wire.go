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

package search

import (
	"context"
	"sync"

	"github.com/ecodeclub/webook/internal/search/ioc"

	"github.com/ecodeclub/webook/internal/cases"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/search/internal/event"

	"github.com/ecodeclub/webook/internal/search/internal/repository"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/search/internal/service"
	"github.com/ecodeclub/webook/internal/search/internal/web"
	"github.com/google/wire"
	"github.com/olivere/elastic/v7"
)

// 初始化adminHandler

func initAdminHandler(es *elastic.Client) *AdminHandler {
	caDAO := ioc.InitAdminCaseDAO(es)
	questionDAO := ioc.InitAdminQuestionDAO(es)
	questionSetDAO := ioc.InitAdminQuestionSetDAO(es)
	skillDAO := ioc.InitAdminSkillDAO(es)
	caRepo := repository.NewCaseRepo(caDAO)
	questionRepo := repository.NewQuestionRepo(questionDAO)
	questionSetRepo := repository.NewQuestionSetRepo(questionSetDAO)
	skillRepo := repository.NewSKillRepo(skillDAO)
	adminSvc := service.NewSearchSvc(questionRepo, questionSetRepo, skillRepo, caRepo)
	return web.NewAdminHandler(adminSvc)
}

// 初始化c端handler
var HandlerSet = wire.NewSet(
	ioc.InitCaseDAO,
	ioc.InitQuestionDAO,
	ioc.InitQuestionSetDAO,
	ioc.InitSkillDAO,
	repository.NewCaseRepo,
	repository.NewQuestionRepo,
	repository.NewQuestionSetRepo,
	repository.NewSKillRepo,
	service.NewSearchSvc,
	web.NewHandler)

// 初始化syncSvc
var SyncSvcSet = wire.NewSet(
	InitAnyRepo,
	InitSyncSvc,
)

func InitModule(es *elastic.Client, q mq.MQ, caModule *cases.Module) (*Module, error) {
	wire.Build(
		initAdminHandler,
		wire.FieldsOf(new(*cases.Module), "ExamineSvc"),
		HandlerSet,
		InitSyncSvc,
		initSyncConsumer,
		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
}

var daoOnce = sync.Once{}

func InitIndexOnce(es *elastic.Client) {
	daoOnce.Do(func() {
		err := dao.InitES(es)
		if err != nil {
			panic(err)
		}
	})
}

func InitAnyRepo(es *elastic.Client) repository.AnyRepo {
	InitIndexOnce(es)
	anyDAO := dao.NewAnyEsDAO(es)
	anyRepo := repository.NewAnyRepo(anyDAO)
	return anyRepo
}

func InitSyncSvc(es *elastic.Client) service.SyncService {
	anyRepo := InitAnyRepo(es)
	return service.NewSyncSvc(anyRepo)
}

func initSyncConsumer(svc service.SyncService, q mq.MQ) *event.SyncConsumer {
	c, err := event.NewSyncConsumer(svc, q)
	if err != nil {
		panic(err)
	}
	c.Start(context.Background())
	return c
}

type SearchService = service.SearchService
type SyncService = service.SyncService
type Handler = web.Handler
type AdminHandler = web.AdminHandler
