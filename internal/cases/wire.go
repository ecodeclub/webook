//go:build wireinject

package cases

import (
	"sync"

	"github.com/ecodeclub/webook/internal/interactive"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/cases/internal/event"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"

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
	q mq.MQ) (*Module, error) {
	wire.Build(InitCaseDAO,
		repository.NewCaseRepo,
		event.NewSyncEventProducer,
		event.NewInteractiveEventProducer,
		service.NewService,
		web.NewHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
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

type Handler = web.Handler
type Service = service.Service
type Case = domain.Case
