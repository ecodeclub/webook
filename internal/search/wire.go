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

func InitModule(es *elastic.Client, q mq.MQ, caModule *cases.Module) (*Module, error) {
	wire.Build(
		InitSearchSvc,
		InitSyncSvc,
		initSyncConsumer,
		wire.FieldsOf(new(*cases.Module), "ExamineSvc"),
		web.NewHandler,
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

func InitRepo(es *elastic.Client) (repository.CaseRepo, repository.QuestionRepo, repository.QuestionSetRepo, repository.SkillRepo) {
	InitIndexOnce(es)
	questionDao := dao.NewQuestionDAO(es)
	caseDao := dao.NewCaseElasticDAO(es)
	questionSetDao := dao.NewQuestionSetDAO(es)
	skillDao := dao.NewSkillElasticDAO(es)
	questionRepo := repository.NewQuestionRepo(questionDao)
	caseRepo := repository.NewCaseRepo(caseDao)
	questionSetRepo := repository.NewQuestionSetRepo(questionSetDao)
	skillRepo := repository.NewSKillRepo(skillDao)
	return caseRepo, questionRepo, questionSetRepo, skillRepo
}
func InitAnyRepo(es *elastic.Client) repository.AnyRepo {
	InitIndexOnce(es)
	anyDAO := dao.NewAnyEsDAO(es)
	anyRepo := repository.NewAnyRepo(anyDAO)
	return anyRepo
}

func InitSearchSvc(es *elastic.Client) service.SearchService {
	caseRepo, questionRepo, questionSetRepo, skillRepo := InitRepo(es)
	return service.NewSearchSvc(questionRepo, questionSetRepo, skillRepo, caseRepo)
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
