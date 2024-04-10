// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package feedback

import (
	"sync"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/feedback/internal/service"
	"github.com/ecodeclub/webook/internal/feedback/internal/web"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

// Injectors from wire.go:

func InitHandler(db *gorm.DB, ec ecache.Cache) (*web.Handler, error) {
	feedbackDAO := InitFeedbackDAO(db)
	feedBackRepo := repository.NewFeedBackRepo(feedbackDAO)
	serviceService := service.NewService(feedBackRepo)
	handler := web.NewHandler(serviceService)
	return handler, nil
}

// wire.go:

var daoOnce = sync.Once{}

func InitTableOnce(db *gorm.DB) {
	daoOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
}

func InitFeedbackDAO(db *egorm.Component) dao.FeedbackDAO {
	InitTableOnce(db)
	return dao.NewFeedbackDAO(db)
}

type Handler = web.Handler
