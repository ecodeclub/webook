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
	"context"
	"sync"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/order/internal/domain"
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

type (
	Handler               = web.Handler
	Service               = service.Service
	CloseTimeoutOrdersJob = job.CloseTimeoutOrdersJob
	Order                 = domain.Order
	OrderStatus           = domain.OrderStatus
	Payment               = domain.Payment
)

const (
	StatusInit       = domain.StatusInit
	StatusProcessing = domain.StatusProcessing
	StatusSuccess    = domain.StatusSuccess
	StatusFailed     = domain.StatusFailed
)

var HandlerSet = wire.NewSet(
	sequencenumber.NewGenerator,
	web.NewHandler)

func InitModule(db *egorm.Component, cache ecache.Cache, q mq.MQ, pm *payment.Module, ppm *product.Module, cm *credit.Module) (*Module, error) {
	wire.Build(
		wire.Struct(new(Module), "*"),
		InitService,
		InitHandler,
		initCompleteOrderConsumer,
		initCloseExpiredOrdersJob,
	)
	return new(Module), nil
}

func InitHandler(cache ecache.Cache, svc service.Service, pm *payment.Module, ppm *product.Module, cm *credit.Module) *Handler {
	wire.Build(
		wire.FieldsOf(new(*payment.Module), "Svc"),
		wire.FieldsOf(new(*product.Module), "Svc"),
		wire.FieldsOf(new(*credit.Module), "Svc"),
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

func initCompleteOrderConsumer(svc service.Service, q mq.MQ) *event.PaymentConsumer {
	consumer, err := event.NewPaymentConsumer(svc, q)
	if err != nil {
		panic(err)
	}
	consumer.Start(context.Background())
	return consumer
}

func initCloseExpiredOrdersJob(svc service.Service) *CloseTimeoutOrdersJob {
	minutes := int64(30)
	seconds := int64(10)
	limit := 100
	return job.NewCloseTimeoutOrdersJob(svc, minutes, seconds, limit)
}
