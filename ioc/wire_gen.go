// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package ioc

import (
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/bff"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/cos"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/feedback"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/label"
	"github.com/ecodeclub/webook/internal/marketing"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/pkg/middleware"
	"github.com/ecodeclub/webook/internal/product"
	"github.com/ecodeclub/webook/internal/project"
	"github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/recon"
	"github.com/ecodeclub/webook/internal/roadmap"
	"github.com/ecodeclub/webook/internal/search"
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
	localActiveLimit := initLocalActiveLimiterBuilder()
	permissionModule, err := permission.InitModule(db, mq)
	if err != nil {
		return nil, err
	}
	serviceService := permissionModule.Svc
	checkPermissionMiddlewareBuilder := middleware.NewCheckPermissionMiddlewareBuilder(serviceService)
	interactiveModule, err := interactive.InitModule(db, mq)
	if err != nil {
		return nil, err
	}
	cache := InitCache(cmdable)
	creditModule, err := credit.InitModule(db, mq, cache)
	if err != nil {
		return nil, err
	}
	aiModule, err := ai.InitModule(db, creditModule)
	if err != nil {
		return nil, err
	}
	baguwenModule, err := baguwen.InitModule(db, interactiveModule, cache, permissionModule, aiModule, mq)
	if err != nil {
		return nil, err
	}
	handler := baguwenModule.Hdl
	examineHandler := baguwenModule.ExamineHdl
	questionSetHandler := baguwenModule.QsHdl
	webHandler := label.InitHandler(db)
	handler2 := InitUserHandler(db, cache, mq, module, permissionModule)
	config := InitCosConfig()
	handler3 := cos.InitHandler(config)
	casesModule, err := cases.InitModule(db, interactiveModule, aiModule, mq)
	if err != nil {
		return nil, err
	}
	handler4 := casesModule.Hdl
	handler5, err := skill.InitHandler(db, cache, baguwenModule, casesModule, mq)
	if err != nil {
		return nil, err
	}
	handler6, err := feedback.InitHandler(db, mq)
	if err != nil {
		return nil, err
	}
	productModule, err := product.InitModule(db, mq)
	if err != nil {
		return nil, err
	}
	handler7 := productModule.Hdl
	paymentModule, err := payment.InitModule(db, mq, cache, creditModule)
	if err != nil {
		return nil, err
	}
	orderModule, err := order.InitModule(db, cache, mq, paymentModule, productModule, creditModule)
	if err != nil {
		return nil, err
	}
	handler8 := orderModule.Hdl
	projectModule, err := project.InitModule(db, interactiveModule, permissionModule, mq)
	if err != nil {
		return nil, err
	}
	handler9 := projectModule.Hdl
	handler10 := creditModule.Hdl
	handler11 := paymentModule.Hdl
	marketingModule, err := marketing.InitModule(db, mq, cache, orderModule, productModule)
	if err != nil {
		return nil, err
	}
	handler12 := marketingModule.Hdl
	handler13 := interactiveModule.Hdl
	client := InitES()
	searchModule, err := search.InitModule(client, mq, casesModule)
	if err != nil {
		return nil, err
	}
	handler14 := searchModule.Hdl
	roadmapModule := roadmap.InitModule(db, baguwenModule)
	handler15 := roadmapModule.Hdl
	bffModule, err := bff.InitModule(interactiveModule, casesModule, baguwenModule)
	if err != nil {
		return nil, err
	}
	handler16 := bffModule.Hdl
	caseSetHandler := casesModule.CsHdl
	webExamineHandler := casesModule.ExamineHdl
	component := initGinxServer(provider, checkMembershipMiddlewareBuilder, localActiveLimit, checkPermissionMiddlewareBuilder, handler, examineHandler, questionSetHandler, webHandler, handler2, handler3, handler4, handler5, handler6, handler7, handler8, handler9, handler10, handler11, handler12, handler13, handler14, handler15, handler16, caseSetHandler, webExamineHandler)
	adminHandler := projectModule.AdminHdl
	webAdminHandler := roadmapModule.AdminHdl
	adminHandler2 := baguwenModule.AdminHdl
	adminQuestionSetHandler := baguwenModule.AdminSetHdl
	adminCaseHandler := casesModule.AdminHandler
	adminCaseSetHandler := casesModule.AdminSetHandler
	adminHandler3 := marketingModule.AdminHdl
	adminServer := InitAdminServer(adminHandler, webAdminHandler, adminHandler2, adminQuestionSetHandler, adminCaseHandler, adminCaseSetHandler, adminHandler3)
	closeTimeoutOrdersJob := orderModule.CloseTimeoutOrdersJob
	closeTimeoutLockedCreditsJob := creditModule.CloseTimeoutLockedCreditsJob
	syncWechatOrderJob := paymentModule.SyncWechatOrderJob
	reconModule, err := recon.InitModule(orderModule, paymentModule, creditModule)
	if err != nil {
		return nil, err
	}
	syncPaymentAndOrderJob := reconModule.SyncPaymentAndOrderJob
	v := initCronJobs(closeTimeoutOrdersJob, closeTimeoutLockedCreditsJob, syncWechatOrderJob, syncPaymentAndOrderJob)
	knowledgeJobStarter := baguwenModule.KnowledgeJobStarter
	v2 := initJobs(knowledgeJobStarter)
	app := &App{
		Web:   component,
		Admin: adminServer,
		Crons: v,
		Jobs:  v2,
	}
	return app, nil
}

// wire.go:

var BaseSet = wire.NewSet(InitDB, InitCache, InitES, InitRedis, InitMQ, InitCosConfig)
