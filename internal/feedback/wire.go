//go:build wireinject

package feedback

import (
	"sync"

	"github.com/ecodeclub/webook/internal/feedback/internal/repository"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/feedback/internal/service"
	"github.com/ecodeclub/webook/internal/feedback/internal/web"

	"github.com/ecodeclub/ecache"

	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitHandler(db *egorm.Component, ec ecache.Cache) (*Handler, error) {
	wire.Build(
		InitFeedBackDAO,
		repository.NewFeedBackRepo,
		service.NewService,
		web.NewHandler,
	)
	return new(Handler), nil
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

func InitFeedBackDAO(db *egorm.Component) dao.FeedBackDAO {
	InitTableOnce(db)
	return dao.NewFeedBackDAO(db)
}

type Handler = web.Handler
