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
	"os"

	"github.com/ecodeclub/ginx/session"

	"github.com/ecodeclub/webook/internal/member"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/permission"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/question/internal/event"
	"github.com/ecodeclub/webook/internal/question/internal/job"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"github.com/ecodeclub/webook/internal/question/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/ecodeclub/webook/internal/question/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitModule(p event.SyncDataToSearchEventProducer,
	knowledgeBaseP event.KnowledgeBaseEventProducer,
	intrModule *interactive.Module,
	permModule *permission.Module,
	aiModule *ai.Module,
	sp session.Provider,
	memberModule *member.Module,
) (*baguwen.Module, error) {
	wire.Build(
		testioc.BaseSet,
		moduleSet,
		event.NewInteractiveEventProducer,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*permission.Module), "Svc"),
		wire.FieldsOf(new(*member.Module), "Svc"),
		wire.FieldsOf(new(*ai.Module), "Svc", "KnowledgeBaseSvc"),
	)
	return new(baguwen.Module), nil
}

var moduleSet = wire.NewSet(baguwen.InitQuestionDAO,
	cache.NewQuestionECache,
	repository.NewCacheRepository,
	service.NewService,
	web.NewHandler,
	web.NewAdminHandler,
	initKnowledgeJobStarter,
	web.NewAdminQuestionSetHandler,
	baguwen.ExamineHandlerSet,
	baguwen.InitQuestionSetDAO,
	repository.NewQuestionSetRepository,
	service.NewQuestionSetService,
	web.NewQuestionSetHandler,
	initKnowledgeBaseSvc,
	web.NewKnowledgeBaseHandler,
	wire.Struct(new(baguwen.Module), "*"),
)

func initKnowledgeJobStarter(svc service.Service) *job.KnowledgeJobStarter {
	return job.NewKnowledgeJobStarter(svc, os.TempDir())
}

func initKnowledgeBaseSvc(svc ai.KnowledgeBaseService, queRepo repository.Repository) service.QuestionKnowledgeBase {
	return service.NewQuestionKnowledgeBase("knowledge_id", queRepo, svc)
}
