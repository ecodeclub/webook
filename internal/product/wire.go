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

package product

import (
	"context"
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/product/internal/event"

	"github.com/ecodeclub/webook/internal/product/internal/domain"
	"github.com/ecodeclub/webook/internal/product/internal/repository"
	"github.com/ecodeclub/webook/internal/product/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/product/internal/service"
	"github.com/ecodeclub/webook/internal/product/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

type (
	Handler = web.Handler
	Service = service.Service
	SKU     = domain.SKU
	SPU     = domain.SPU
	Status  = domain.Status
)

const (
	StatusOffShelf    = domain.StatusOffShelf
	StatusOnShelf     = domain.StatusOnShelf
	SaleTypeUnlimited = domain.SaleTypeUnlimited
)

var ServiceSet = wire.NewSet(
	InitTablesOnce,
	repository.NewProductRepository,
	service.NewService)

var HandlerSet = wire.NewSet(
	InitService,
	web.NewHandler)

func InitModule(db *egorm.Component, cmq mq.MQ) (*Module, error) {
	wire.Build(HandlerSet, InitConsumer, wire.Struct(new(Module), "*"))
	return new(Module), nil
}

func InitHandler(db *egorm.Component) *Handler {
	wire.Build(HandlerSet)
	return new(Handler)
}

func InitService(db *egorm.Component) Service {
	wire.Build(ServiceSet)
	return nil
}

func InitConsumer(svc service.Service, cmq mq.MQ) *event.ProductConsumer {
	consumer, err := event.NewProductConsumer(svc, cmq)
	if err != nil {
		panic(err)
	}
	consumer.Start(context.Background())
	return consumer
}

var once = &sync.Once{}

func InitTablesOnce(db *egorm.Component) dao.ProductDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
	})
	return dao.NewProductGORMDAO(db)
}
