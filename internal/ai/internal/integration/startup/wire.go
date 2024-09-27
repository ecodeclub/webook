//go:build wireinject

package startup

import (
	"sync"

	hdlmocks "github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/mocks"

	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/config"
	aicredit "github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/credit"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/log"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/record"

	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitModule(db *egorm.Component,
	hdl *hdlmocks.MockHandler,
	creditSvc *credit.Module) (*ai.Module, error) {
	wire.Build(
		llm.NewLLMService,
		repository.NewLLMLogRepo,
		repository.NewLLMCreditLogRepo,
		repository.NewCachedConfigRepository,

		InitLLMCreditLogDAO,
		dao.NewGORMLLMLogDAO,
		dao.NewGORMConfigDAO,

		config.NewBuilder,
		log.NewHandler,
		record.NewHandler,
		aicredit.NewHandlerBuilder,

		ai.InitCommonHandlers,
		InitRootHandler,

		wire.Struct(new(ai.Module), "*"),
		wire.FieldsOf(new(*credit.Module), "Svc"),
	)
	return new(ai.Module), nil
}

func InitRootHandler(common []handler.Builder, hdl *hdlmocks.MockHandler) handler.Handler {
	return handler.NewCompositionHandler(common, hdl)
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
