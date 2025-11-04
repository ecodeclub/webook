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
	"fmt"

	"github.com/ecodeclub/webook/internal/kbase"
	"github.com/ecodeclub/webook/internal/kbase/internal/domain"
	"github.com/ecodeclub/webook/internal/kbase/internal/service"
	"github.com/ecodeclub/webook/internal/kbase/internal/service/syncer"
	"github.com/ecodeclub/webook/internal/kbase/internal/web"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/roadmap"
	"github.com/google/wire"
)

func InitModule(queModule *baguwen.Module, rdModule *roadmap.Module, svc service.Service) *kbase.Module {
	wire.Build(
		wire.FieldsOf(new(*baguwen.Module), "Svc"),
		wire.FieldsOf(new(*roadmap.Module), "AdminSvc"),
		initSyncerMap,
		service.NewSyncService,
		web.NewAdminHandler,
		wire.Struct(new(kbase.Module), "*"),
	)
	return new(kbase.Module)
}

func initSyncerMap(baguwenSvc baguwen.Service, rdSvc roadmap.AdminService, svc service.Service) map[string]service.Syncer {
	questionIndexName := fmt.Sprintf("%s_index", domain.BizQuestion)
	questionRelIndexName := fmt.Sprintf("%s_index", domain.BizQuestionRel)
	batchSize := 100
	return map[string]service.Syncer{
		domain.BizQuestion: syncer.NewQuestionSyncer(questionIndexName,
			batchSize, baguwenSvc, svc),
		domain.BizQuestionRel: syncer.NewQuestionRelSyncer(questionRelIndexName,
			batchSize, rdSvc, svc),
	}
}
