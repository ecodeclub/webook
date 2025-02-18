//go:build wireinject

package cases

import (
	"sync"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/cache"

	"github.com/ecodeclub/ginx/session"

	"github.com/ecodeclub/webook/internal/member"

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
	memberModule *member.Module,
	sp session.Provider,
	redisCache ecache.Cache,
	q mq.MQ) (*Module, error) {
	wire.Build(InitCaseDAO,
		dao.NewCaseSetDAO,
		dao.NewGORMExamineDAO,
		cache.NewCaseCache,
		repository.NewCaseRepo,
		repository.NewCaseSetRepo,
		repository.NewCachedExamineRepository,
		event.NewSyncEventProducer,
		event.NewInteractiveEventProducer,
		service.NewCaseSetService,
		service.NewService,
		service.NewLLMExamineService,
		InitKnowledgeBaseEvt,
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
		wire.FieldsOf(new(*member.Module), "Svc"),
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
