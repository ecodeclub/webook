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

package order

import (
	"sync"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/order/internal/repository"
	"github.com/ecodeclub/webook/internal/order/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/order/internal/service"
	"github.com/ecodeclub/webook/internal/order/internal/web"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/ecodeclub/webook/internal/product"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

var HandlerSet = wire.NewSet(
	InitTablesOnce,
	repository.NewRepository,
	service.NewService,
	sequencenumber.NewGenerator,
	web.NewHandler)

func InitHandler(db *egorm.Component, paymentSvc payment.Service, productSvc product.Service, creditSvc credit.Service, cache ecache.Cache) *Handler {
	wire.Build(HandlerSet)
	return new(Handler)
}

var once = &sync.Once{}

func InitTablesOnce(db *egorm.Component) dao.OrderDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
	})
	return dao.NewOrderGORMDAO(db)
}

type Handler = web.Handler
