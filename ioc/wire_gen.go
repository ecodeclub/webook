// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package ioc

import (
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/cos"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/feedback"
	"github.com/ecodeclub/webook/internal/label"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/pkg/middleware"
	"github.com/ecodeclub/webook/internal/product"
	"github.com/ecodeclub/webook/internal/project"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/skill"
	"github.com/google/wire"
)

// Injectors from wire.go:

func InitApp() (*App, error) {
	cmdable := InitRedis()
	provider := InitSession(cmdable)
	db := InitDB()
	mq := InitMQ()
	module, err := member.InitModule(db, mq)
	if err != nil {
		return nil, err
	}
	service := module.Svc
	checkMembershipMiddlewareBuilder := middleware.NewCheckMembershipMiddlewareBuilder(service)
	cache := InitCache(cmdable)
	baguwenModule, err := baguwen.InitModule(db, cache)
	if err != nil {
		return nil, err
	}
	handler := baguwenModule.Hdl
	questionSetHandler := baguwenModule.QsHdl
	webHandler := label.InitHandler(db)
	handler2 := InitUserHandler(db, cache, mq, module)
	config := InitCosConfig()
	handler3 := cos.InitHandler(config)
	casesModule, err := cases.InitModule(db, cache)
	if err != nil {
		return nil, err
	}
	handler4 := casesModule.Hdl
	handler5, err := skill.InitHandler(db, cache, baguwenModule, casesModule)
	if err != nil {
		return nil, err
	}
	handler6, err := feedback.InitHandler(db, mq)
	if err != nil {
		return nil, err
	}
	productModule, err := product.InitModule(db)
	if err != nil {
		return nil, err
	}
	handler7 := productModule.Hdl
	creditModule, err := credit.InitModule(db, mq, cache)
	if err != nil {
		return nil, err
	}
	paymentModule, err := payment.InitModule(db, mq, cache, creditModule)
	if err != nil {
		return nil, err
	}
	orderModule, err := order.InitModule(db, cache, mq, paymentModule, productModule, creditModule)
	if err != nil {
		return nil, err
	}
	handler8 := orderModule.Hdl
	projectModule := project.InitModule()
	handler9 := projectModule.Hdl
	handler10 := creditModule.Hdl
	handler11 := paymentModule.Hdl
	component := initGinxServer(provider, checkMembershipMiddlewareBuilder, handler, questionSetHandler, webHandler, handler2, handler3, handler4, handler5, handler6, handler7, handler8, handler9, handler10, handler11)
	adminHandler := projectModule.AdminHdl
	adminServer := InitAdminServer(adminHandler)
	closeTimeoutOrdersJob := orderModule.CloseTimeoutOrdersJob
	closeTimeoutLockedCreditsJob := creditModule.CloseTimeoutLockedCreditsJob
	syncWechatOrderJob := paymentModule.SyncWechatOrderJob
	v := initCronJobs(closeTimeoutOrdersJob, closeTimeoutLockedCreditsJob, syncWechatOrderJob)
	app := &App{
		Web:   component,
		Admin: adminServer,
		Jobs:  v,
	}
	return app, nil
}

// wire.go:

var BaseSet = wire.NewSet(InitDB, InitCache, InitRedis, InitMQ, InitCosConfig)
