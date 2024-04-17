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
	"sync"

	"github.com/ecodeclub/webook/internal/product/internal/domain"
	"github.com/ecodeclub/webook/internal/product/internal/repository"
	"github.com/ecodeclub/webook/internal/product/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/product/internal/service"
	"github.com/ecodeclub/webook/internal/product/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

var ServiceSet = wire.NewSet(
	InitTablesOnce,
	repository.NewProductRepository,
	service.NewService)

var HandlerSet = wire.NewSet(
	InitService,
	web.NewHandler)

func InitModule(db *egorm.Component) *Module {
	wire.Build(HandlerSet, wire.Struct(new(Module), "*"))
	return new(Module)
}

func InitHandler(db *egorm.Component) *Handler {
	wire.Build(HandlerSet)
	return new(Handler)
}

func InitService(db *egorm.Component) Service {
	wire.Build(ServiceSet)
	return nil
}

var once = &sync.Once{}

func InitTablesOnce(db *egorm.Component) dao.ProductDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
	})
	return dao.NewProductGORMDAO(db)
}

type Handler = web.Handler

type Service = service.Service

type SKU = domain.SKU
type SPU = domain.SPU
type Status = domain.Status

const StatusOffShelf = domain.StatusOffShelf
const StatusOnShelf = domain.StatusOnShelf
const SaleTypeUnlimited = domain.SaleTypeUnlimited
