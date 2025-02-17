//go:build wireinject

package review

import (
	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/review/internal/event"
	"github.com/ecodeclub/webook/internal/review/internal/repository"
	"github.com/ecodeclub/webook/internal/review/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/review/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/review/internal/service"
	"github.com/ecodeclub/webook/internal/review/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

func InitModule(db *egorm.Component,
	interSvc *interactive.Module,
	q mq.MQ,
	sp session.Provider,
	ec ecache.Cache,
) *Module {
	wire.Build(
		initReviewDao,
		initIntrProducer,
		repository.NewReviewRepo,
		service.NewReviewSvc,
		cache.NewReviewCache,
		web.NewHandler,
		web.NewAdminHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.Struct(new(Module), "*"),
	)
	return new(Module)
}

func initReviewDao(db *egorm.Component) dao.ReviewDAO {
	err := dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return dao.NewReviewDAO(db)
}

func initIntrProducer(q mq.MQ) event.InteractiveEventProducer {
	producer, err := event.NewInteractiveEventProducer(q)
	if err != nil {
		panic(err)
	}
	return producer
}
