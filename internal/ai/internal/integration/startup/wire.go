//go:build wireinject

package startup

import (
	"sync"

	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt"
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler"
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler/biz"
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler/config"
	aicredit "github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler/credit"
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler/log"
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler/record"

	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitModule(db *egorm.Component,
	hdl handler.Handler,
	creditSvc *credit.Module) (*ai.Module, error) {
	wire.Build(
		gpt.NewGPTService,
		repository.NewGPTLogRepo,
		repository.NewGPTCreditLogRepo,
		repository.NewCachedConfigRepository,

		InitGPTCreditLogDAO,
		dao.NewGORMGPTLogDAO,
		dao.NewGORMConfigDAO,

		config.NewBuilder,
		log.NewHandler,
		record.NewHandler,
		aicredit.NewHandlerBuilder,

		ai.InitCommonHandlers,
		InitHandlerFacade,

		wire.Struct(new(ai.Module), "*"),
		wire.FieldsOf(new(*credit.Module), "Svc"),
	)
	return new(ai.Module), nil
}

func InitHandlerFacade(common []handler.Builder, gpt handler.Handler) *biz.FacadeHandler {
	que := ai.InitQuestionExamineHandler(common, gpt)
	return biz.NewHandler(map[string]handler.Handler{
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

func InitGPTCreditLogDAO(db *egorm.Component) dao.GPTCreditDAO {
	InitTableOnce(db)
	return dao.NewGPTCreditLogDAO(db)
}
