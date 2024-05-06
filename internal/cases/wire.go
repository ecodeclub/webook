//go:build wireinject

package cases

import (
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/cases/internal/event"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/ecodeclub/webook/internal/cases/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitModule(db *egorm.Component, ec ecache.Cache, q mq.MQ) (*Module, error) {
	wire.Build(InitCaseDAO,
		cache.NewCaseCache,
		repository.NewCaseRepo,
		event.NewSyncEventProducer,
		NewService,
		web.NewHandler,
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

func NewService(repo repository.CaseRepo, producer event.SyncEventProducer) Service {
	return service.NewService(repo, producer)
}

func InitCaseDAO(db *egorm.Component) dao.CaseDAO {
	InitTableOnce(db)
	return dao.NewCaseDao(db)
}

type Handler = web.Handler
type Service = service.Service
type Case = domain.Case
