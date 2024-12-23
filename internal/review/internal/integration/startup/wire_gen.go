// Code generated by Wire. DO NOT EDIT.

//go:generate go run -mod=mod github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package startup

import (
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/review"
	"github.com/ecodeclub/webook/internal/review/internal/repository"
	"github.com/ecodeclub/webook/internal/review/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/review/internal/service"
	"github.com/ecodeclub/webook/internal/review/internal/web"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

// Injectors from wire.go:

func InitModule(db *gorm.DB, interSvc *interactive.Module) *review.Module {
	reviewDAO := initReviewDao(db)
	reviewRepo := repository.NewReviewRepo(reviewDAO)
	reviewSvc := service.NewReviewSvc(reviewRepo)
	serviceService := interSvc.Svc
	handler := web.NewHandler(reviewSvc, serviceService)
	adminHandler := web.NewAdminHandler(reviewSvc)
	module := &review.Module{
		Hdl:      handler,
		AdminHdl: adminHandler,
	}
	return module
}

// wire.go:

func initReviewDao(db *egorm.Component) dao.ReviewDAO {
	err := dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return dao.NewReviewDAO(db)
}
