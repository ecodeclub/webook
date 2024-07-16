//go:build wireinject

package ai

import (
	"sync"

	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt"
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

func InitModule(db *egorm.Component, creditSvc *credit.Module) (*Module, error) {
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

		InitHandlerFacade,
		InitCommonHandlers,
		InitZhipu,

		wire.Struct(new(Module), "*"),
		wire.FieldsOf(new(*credit.Module), "Svc"),
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

func InitGPTCreditLogDAO(db *egorm.Component) dao.GPTCreditDAO {
	InitTableOnce(db)
	return dao.NewGPTCreditLogDAO(db)
}
