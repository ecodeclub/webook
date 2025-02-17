//go:build wireinject

package startup

import (
	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/review"
	"github.com/ecodeclub/webook/internal/review/internal/event"
	"github.com/ecodeclub/webook/internal/review/internal/repository"
	"github.com/ecodeclub/webook/internal/review/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/review/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/review/internal/service"
	"github.com/ecodeclub/webook/internal/review/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

func InitModule(db *egorm.Component, interSvc *interactive.Module, q mq.MQ, ec ecache.Cache, sp session.Provider) *review.Module {
	wire.Build(
		initReviewDao,
		initIntrProducer,
		repository.NewReviewRepo,
		cache.NewReviewCache,
		service.NewReviewSvc,
		web.NewHandler,
		web.NewAdminHandler,
		wire.Struct(new(review.Module), "*"),
		wire.FieldsOf(new(*interactive.Module), "Svc"),
	)
	return new(review.Module)
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
