// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package startup

import (
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/biz"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/config"
	credit2 "github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/credit"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/log"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/record"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"sync"
)

// Injectors from wire.go:

func InitModule(db *gorm.DB, hdl handler.Handler, creditSvc *credit.Module) (*ai.Module, error) {
	handlerBuilder := log.NewHandler()
	configDAO := dao.NewGORMConfigDAO(db)
	configRepository := repository.NewCachedConfigRepository(configDAO)
	configHandlerBuilder := config.NewBuilder(configRepository)
	service := creditSvc.Svc
	llmCreditDAO := InitLLMCreditLogDAO(db)
	llmCreditLogRepo := repository.NewLLMCreditLogRepo(llmCreditDAO)
	creditHandlerBuilder := credit2.NewHandlerBuilder(service, llmCreditLogRepo)
	llmRecordDAO := dao.NewGORMLLMLogDAO(db)
	llmLogRepo := repository.NewLLMLogRepo(llmRecordDAO)
	recordHandlerBuilder := record.NewHandler(llmLogRepo)
	v := ai.InitCommonHandlers(handlerBuilder, configHandlerBuilder, creditHandlerBuilder, recordHandlerBuilder)
	facadeHandler := InitHandlerFacade(v, hdl)
	llmService := llm.NewLLMService(facadeHandler)
	module := &ai.Module{
		Svc: llmService,
	}
	return module, nil
}

// wire.go:

func InitHandlerFacade(common []handler.Builder, llm2 handler.Handler) *biz.FacadeHandler {
	que := ai.InitQuestionExamineHandler(common, llm2)
	ca := ai.InitCaseExamineHandler(common, llm2)
	return biz.NewHandler(map[string]handler.Handler{
		ca.Biz():  ca,
		que.Biz(): que,
	})
}

var daoOnce = sync.Once{}

func InitTableOnce(db *gorm.DB) {
	daoOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
}

func InitLLMCreditLogDAO(db *egorm.Component) dao.LLMCreditDAO {
	InitTableOnce(db)
	return dao.NewLLMCreditLogDAO(db)
}
