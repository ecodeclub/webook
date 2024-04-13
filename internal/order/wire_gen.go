// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package order

import (
	"sync"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/mq-api"
	service3 "github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/order/internal/event"
	"github.com/ecodeclub/webook/internal/order/internal/job"
	"github.com/ecodeclub/webook/internal/order/internal/repository"
	"github.com/ecodeclub/webook/internal/order/internal/repository/dao"
	service4 "github.com/ecodeclub/webook/internal/order/internal/service"
	"github.com/ecodeclub/webook/internal/order/internal/web"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	service2 "github.com/ecodeclub/webook/internal/product"
	"github.com/google/wire"
	"gorm.io/gorm"
)

// Injectors from wire.go:

func InitModule(db *gorm.DB, cache ecache.Cache, q mq.MQ, paymentSvc payment.Service, productSvc service2.Service, creditSvc service3.Service) (*Module, error) {
	serviceService := InitService(db)
	handler := InitHandler(cache, serviceService, paymentSvc, productSvc, creditSvc)
	completeOrderConsumer := initCompleteOrderConsumer(serviceService, q)
	module := &Module{
		Hdl: handler,
		c:   completeOrderConsumer,
	}
	return module, nil
}

func InitHandler(cache ecache.Cache, svc2 service4.Service, paymentSvc payment.Service, productSvc service2.Service, creditSvc service3.Service) *web.Handler {
	generator := sequencenumber.NewGenerator()
	handler := web.NewHandler(svc2, paymentSvc, productSvc, creditSvc, generator, cache)
	return handler
}

// wire.go:

type Handler = web.Handler

type Service = service4.Service

type CloseExpiredOrdersJob = job.CloseExpiredOrdersJob

var HandlerSet = wire.NewSet(sequencenumber.NewGenerator, web.NewHandler)

var (
	once = &sync.Once{}
	svc  service4.Service
)

func InitService(db *gorm.DB) service4.Service {
	once.Do(func() {
		_ = dao.InitTables(db)
		orderDAO := dao.NewOrderGORMDAO(db)
		orderRepository := repository.NewRepository(orderDAO)
		svc = service4.NewService(orderRepository)
	})
	return svc
}

func initCompleteOrderConsumer(svc2 service4.Service, q mq.MQ) *event.CompleteOrderConsumer {
	consumer, err := event.NewCompleteOrderConsumer(svc2, q)
	if err != nil {
		panic(err)
	}
	return consumer
}
