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

	"github.com/ecodeclub/webook/internal/product/internal/repository"
	"github.com/ecodeclub/webook/internal/product/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/product/internal/service"
	"github.com/ecodeclub/webook/internal/product/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

var HandlerSet = wire.NewSet(
	repository.NewProductRepository,
	InitTablesOnce,
	service.NewService,
	web.NewHandler)

func InitHandler(db *egorm.Component) *Handler {
	wire.Build(HandlerSet)
	return new(Handler)
}

var once = &sync.Once{}

func InitTablesOnce(db *egorm.Component) dao.ProductDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
	})
	return dao.NewProductGORMDAO(db)
}

type Handler = web.Handler
