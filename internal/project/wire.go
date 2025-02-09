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

package project

import (
	"sync"

	"github.com/ecodeclub/ginx/session"

	"github.com/ecodeclub/webook/internal/permission"

	"github.com/ecodeclub/webook/internal/interactive"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/project/internal/event"

	"github.com/ecodeclub/webook/internal/project/internal/repository"
	"github.com/ecodeclub/webook/internal/project/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/project/internal/service"
	"github.com/ecodeclub/webook/internal/project/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

func InitModule(db *egorm.Component,
	intrModule *interactive.Module,
	permModule *permission.Module,
	q mq.MQ,
	sp session.Provider,
) (*Module, error) {
	wire.Build(
		initSyncToSearchEventProducer,
		initAdminDAO,
		repository.NewProjectAdminRepository,
		service.NewProjectAdminService,
		event.NewSyncProjectToSearchEventProducer,
		event.NewInteractiveEventProducer,
		web.NewAdminHandler,

		dao.NewGORMProjectDAO,
		repository.NewCachedRepository,
		service.NewService,
		web.NewHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*permission.Module), "Svc"),
		wire.Struct(new(Module), "*"))
	return &Module{}, nil
}

var (
	adminDAO     dao.ProjectAdminDAO
	adminDAOOnce sync.Once
)

func initAdminDAO(db *egorm.Component) dao.ProjectAdminDAO {
	adminDAOOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
		adminDAO = dao.NewGORMProjectAdminDAO(db)
	})
	return adminDAO
}

func initSyncToSearchEventProducer(q mq.MQ) mq.Producer {
	res, err := q.Producer(event.SyncTopic)
	if err != nil {
		panic(err)
	}
	return res
}
