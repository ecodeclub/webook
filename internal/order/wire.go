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
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/order/internal/event"
	"github.com/ecodeclub/webook/internal/order/internal/job"
	"github.com/ecodeclub/webook/internal/order/internal/repository"
	"github.com/ecodeclub/webook/internal/order/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/order/internal/service"
	"github.com/ecodeclub/webook/internal/order/internal/web"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/ecodeclub/webook/internal/product"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

type Handler = web.Handler
type Service = service.Service
type CloseExpiredOrdersJob = job.CloseExpiredOrdersJob

var HandlerSet = wire.NewSet(
	sequencenumber.NewGenerator,
	web.NewHandler)

func InitModule(db *egorm.Component, cache ecache.Cache, q mq.MQ, paymentSvc payment.Service, productSvc product.Service, creditSvc credit.Service) (*Module, error) {
	wire.Build(
		wire.Struct(new(Module), "*"),
		InitService,
		InitHandler,
		initCompleteOrderConsumer,
	)
	return new(Module), nil
}

func InitHandler(cache ecache.Cache, svc service.Service, paymentSvc payment.Service, productSvc product.Service, creditSvc credit.Service) *Handler {
	wire.Build(
		sequencenumber.NewGenerator,
		web.NewHandler)
	return new(Handler)
}

var (
	once = &sync.Once{}
	svc  service.Service
)

func InitService(db *gorm.DB) service.Service {
	once.Do(func() {
		_ = dao.InitTables(db)
		orderDAO := dao.NewOrderGORMDAO(db)
		orderRepository := repository.NewRepository(orderDAO)
		svc = service.NewService(orderRepository)
	})
	return svc
}

func initCompleteOrderConsumer(svc service.Service, q mq.MQ) *event.CompleteOrderConsumer {
	consumer, err := event.NewCompleteOrderConsumer(svc, q)
	if err != nil {
		panic(err)
	}
	return consumer
}
