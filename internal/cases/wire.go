//go:build wireinject

package cases

import (
	"sync"

	"github.com/ecodeclub/webook/internal/ai"

	"github.com/ecodeclub/webook/internal/interactive"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/cases/internal/event"

	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitModule(db *egorm.Component,
	intrModule *interactive.Module,
	aiModule *ai.Module,
	q mq.MQ) (*Module, error) {
	wire.Build(InitCaseDAO,
		dao.NewCaseSetDAO,
		dao.NewGORMExamineDAO,
		repository.NewCaseRepo,
		repository.NewCaseSetRepo,
		repository.NewCachedExamineRepository,
		event.NewSyncEventProducer,
		event.NewInteractiveEventProducer,
		service.NewCaseSetService,
		service.NewService,
		service.NewLLMExamineService,
		InitKnowledgeBaseSvc,
		web.NewHandler,
		web.NewAdminCaseSetHandler,
		web.NewExamineHandler,
		web.NewCaseSetHandler,
		web.NewAdminCaseHandler,
		web.NewKnowledgeBaseHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*ai.Module), "Svc", "KnowledgeBaseSvc"),
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

func InitCaseDAO(db *egorm.Component) dao.CaseDAO {
	InitTableOnce(db)
	return dao.NewCaseDao(db)
}
