//go:build wireinject

package ai

import (
	"sync"

	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/ai/internal/service"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/biz"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/config"
	aiCredit "github.com/ecodeclub/webook/internal/ai/internal/service/handler/credit"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt/getter"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt/sdk"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/log"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/response"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/simple"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitModule(db *egorm.Component,
	aisdk sdk.GPTSdk, creditSvc credit.Service) (*Module, error) {
	wire.Build(InitGPTDAO,
		repository.NewGPTLogRepo,
		InitHandlers,
		service.NewGPTService,
		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
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

func InitGPTDAO(db *egorm.Component) dao.GPTLogDAO {
	InitTableOnce(db)
	return dao.NewGPTLogDAO(db)
}

func InitGptHandler(sdk1 sdk.GPTSdk) *gpt.Handler {
	sdkGetter := getter.NewPollingGetter([]sdk.GPTSdk{sdk1})
	gptHandler, err := gpt.NewHandler(sdkGetter)
	if err != nil {
		panic(err)
	}
	return gptHandler
}

func InitHandlers(repo repository.GPTLogRepo, sdk1 sdk.GPTSdk, creditSvc credit.Service) []handler.GptHandler {
	logHandler := log.NewHandler()
	creditHandler := aiCredit.NewHandler(creditSvc, repo)
	gptHandler := InitGptHandler(sdk1)
	configHandler := config.InitHandler()
	simpleHandler := simple.InitHandler(logHandler, creditHandler, gptHandler)
	bizHandler := biz.NewHandler(map[string]handler.GptHandler{
		"simple": simpleHandler,
	})
	responseHandler := response.NewHandler(repo)
	return []handler.GptHandler{
		responseHandler,
		configHandler,
		bizHandler,
	}
}
