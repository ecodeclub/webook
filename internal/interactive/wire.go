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

package interactive

import (
	"context"
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/interactive/internal/events"
	"github.com/ecodeclub/webook/internal/interactive/internal/repository"
	"github.com/ecodeclub/webook/internal/interactive/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/interactive/internal/service"
	"github.com/ecodeclub/webook/internal/interactive/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

var HandlerSet = wire.NewSet(
	InitTablesOnce,
	repository.NewCachedInteractiveRepository,
	service.NewService,
	web.NewHandler)

func InitModule(db *egorm.Component, q mq.MQ) (*Module, error) {
	wire.Build(
		InitTablesOnce,
		repository.NewCachedInteractiveRepository,
		service.NewService,
		initConsumer,
		web.NewHandler,
		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
}

var once = &sync.Once{}

func InitTablesOnce(db *egorm.Component) dao.InteractiveDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
	})
	return dao.NewInteractiveDAO(db)
}

func initConsumer(svc service.InteractiveService, q mq.MQ) *events.Consumer {
	consumer, err := events.NewSyncConsumer(svc, q)
	if err != nil {
		panic(err)
	}
	consumer.Start(context.Background())
	return consumer
}

type Handler = web.Handler

type InteractiveSvc = service.InteractiveService
