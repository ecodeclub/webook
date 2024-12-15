//go:build wireinject

package startup

import (
	"github.com/ecodeclub/webook/internal/review"
	"github.com/ecodeclub/webook/internal/review/internal/repository"
	"github.com/ecodeclub/webook/internal/review/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/review/internal/service"
	"github.com/ecodeclub/webook/internal/review/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

func InitModule(db *egorm.Component) *review.Module {
	wire.Build(
		initReviewDao,
		repository.NewReviewRepo,
		service.NewReviewSvc,
		web.NewHandler,
		web.NewAdminHandler,
		wire.Struct(new(review.Module), "*"),
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
