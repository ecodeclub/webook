// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package startup

import (
	"os"

	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/permission"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/question/internal/event"
	"github.com/ecodeclub/webook/internal/question/internal/job"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"github.com/ecodeclub/webook/internal/question/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/ecodeclub/webook/internal/question/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

// Injectors from wire.go:

func InitModule(p event.SyncDataToSearchEventProducer, intrModule *interactive.Module, permModule *permission.Module, aiModule *ai.Module) (*baguwen.Module, error) {
	db := testioc.InitDB()
	questionDAO := baguwen.InitQuestionDAO(db)
	ecacheCache := testioc.InitCache()
	questionCache := cache.NewQuestionECache(ecacheCache)
	repositoryRepository := repository.NewCacheRepository(questionDAO, questionCache)
	mq := testioc.InitMQ()
	interactiveEventProducer, err := event.NewInteractiveEventProducer(mq)
	if err != nil {
		return nil, err
	}
	serviceService := service.NewService(repositoryRepository, p, interactiveEventProducer)
	questionSetDAO := baguwen.InitQuestionSetDAO(db)
	questionSetRepository := repository.NewQuestionSetRepository(questionSetDAO)
	questionSetService := service.NewQuestionSetService(questionSetRepository, repositoryRepository, interactiveEventProducer, p)
	adminHandler := web.NewAdminHandler(serviceService)
	adminQuestionSetHandler := web.NewAdminQuestionSetHandler(questionSetService)
	service2 := intrModule.Svc
	examineDAO := dao.NewGORMExamineDAO(db)
	examineRepository := repository.NewCachedExamineRepository(examineDAO)
	llmService := aiModule.Svc
	examineService := service.NewLLMExamineService(repositoryRepository, examineRepository, llmService)
	service3 := permModule.Svc
	handler := web.NewHandler(service2, examineService, service3, serviceService)
	questionSetHandler := web.NewQuestionSetHandler(questionSetService, examineService, service2)
	examineHandler := web.NewExamineHandler(examineService)
	knowledgeJobStarter := initKnowledgeJobStarter(serviceService)
	module := &baguwen.Module{
		Svc:                 serviceService,
		SetSvc:              questionSetService,
		AdminHdl:            adminHandler,
		AdminSetHdl:         adminQuestionSetHandler,
		Hdl:                 handler,
		QsHdl:               questionSetHandler,
		ExamineHdl:          examineHandler,
		KnowledgeJobStarter: knowledgeJobStarter,
	}
	return module, nil
}

// wire.go:

var moduleSet = wire.NewSet(baguwen.InitQuestionDAO, cache.NewQuestionECache, repository.NewCacheRepository, service.NewService, web.NewHandler, web.NewAdminHandler, initKnowledgeJobStarter, web.NewAdminQuestionSetHandler, baguwen.ExamineHandlerSet, baguwen.InitQuestionSetDAO, repository.NewQuestionSetRepository, service.NewQuestionSetService, web.NewQuestionSetHandler, wire.Struct(new(baguwen.Module), "*"))

func initKnowledgeJobStarter(svc service.Service) *job.KnowledgeJobStarter {
	return job.NewKnowledgeJobStarter(svc, os.TempDir())
}
