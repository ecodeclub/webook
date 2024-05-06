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
	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/cases/internal/event"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
	"gorm.io/gorm"

	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitHandler(p event.SyncEventProducer) (*web.Handler, error) {
	wire.Build(testioc.BaseSet, initModule,
		wire.FieldsOf(new(*cases.Module), "Hdl"))
	return new(web.Handler), nil
}

func initModule(db *gorm.DB, ec ecache.Cache, p event.SyncEventProducer) (*cases.Module, error) {
	caseDAO := cases.InitCaseDAO(db)
	caseCache := cache.NewCaseCache(ec)
	caseRepo := repository.NewCaseRepo(caseDAO, caseCache)
	service := cases.NewService(caseRepo, p)
	handler := web.NewHandler(service)
	module := &cases.Module{
		Svc: service,
		Hdl: handler,
	}
	return module, nil
}
