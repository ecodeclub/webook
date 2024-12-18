// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package cases

import (
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/cases/internal/event"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

// Injectors from wire.go:

func InitModule(db *gorm.DB, intrModule *interactive.Module, aiModule *ai.Module, q mq.MQ) (*Module, error) {
	caseDAO := InitCaseDAO(db)
	caseRepo := repository.NewCaseRepo(caseDAO)
	interactiveEventProducer, err := event.NewInteractiveEventProducer(q)
	if err != nil {
		return nil, err
	}
	knowledgeBaseEventProducer := InitKnowledgeBaseEvt(q)
	syncEventProducer, err := event.NewSyncEventProducer(q)
	if err != nil {
		return nil, err
	}
	serviceService := service.NewService(caseRepo, interactiveEventProducer, knowledgeBaseEventProducer, syncEventProducer)
	caseSetDAO := dao.NewCaseSetDAO(db)
	caseSetRepository := repository.NewCaseSetRepo(caseSetDAO)
	caseSetService := service.NewCaseSetService(caseSetRepository, caseRepo, interactiveEventProducer)
	examineDAO := dao.NewGORMExamineDAO(db)
	examineRepository := repository.NewCachedExamineRepository(examineDAO)
	llmService := aiModule.Svc
	examineService := service.NewLLMExamineService(caseRepo, examineRepository, llmService)
	service2 := intrModule.Svc
	handler := web.NewHandler(serviceService, examineService, service2)
	adminCaseSetHandler := web.NewAdminCaseSetHandler(caseSetService)
	adminCaseHandler := web.NewAdminCaseHandler(serviceService)
	examineHandler := web.NewExamineHandler(examineService)
	caseSetHandler := web.NewCaseSetHandler(caseSetService, examineService, service2)
	repositoryBaseSvc := aiModule.KnowledgeBaseSvc
	knowledgeBaseService := InitKnowledgeBaseSvc(repositoryBaseSvc, caseRepo)
	knowledgeBaseHandler := web.NewKnowledgeBaseHandler(knowledgeBaseService)
	module := &Module{
		Svc:                  serviceService,
		SetSvc:               caseSetService,
		ExamineSvc:           examineService,
		Hdl:                  handler,
		AdminSetHandler:      adminCaseSetHandler,
		AdminHandler:         adminCaseHandler,
		ExamineHdl:           examineHandler,
		CsHdl:                caseSetHandler,
		KnowledgeBaseHandler: knowledgeBaseHandler,
	}
	return module, nil
}

// wire.go:

var daoOnce = sync.Once{}

func InitTableOnce(db *gorm.DB) {
	daoOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
}

func InitCaseDAO(db *egorm.Component) dao.CaseDAO {
	InitTableOnce(db)
	return dao.NewCaseDao(db)
}
