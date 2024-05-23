// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package project

import (
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/project/internal/event"
	"github.com/ecodeclub/webook/internal/project/internal/repository"
	"github.com/ecodeclub/webook/internal/project/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/project/internal/service"
	"github.com/ecodeclub/webook/internal/project/internal/web"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

// Injectors from wire.go:

func InitModule(db *gorm.DB, intrModule *interactive.Module, permModule *permission.Module, q mq.MQ) (*Module, error) {
	projectAdminDAO := initAdminDAO(db)
	projectAdminRepository := repository.NewProjectAdminRepository(projectAdminDAO)
	producer := initSyncToSearchEventProducer(q)
	syncProjectToSearchEventProducer := event.NewSyncProjectToSearchEventProducer(producer)
	projectDAO := dao.NewGORMProjectDAO(db)
	repositoryRepository := repository.NewCachedRepository(projectDAO)
	projectAdminService := service.NewProjectAdminService(projectAdminRepository, syncProjectToSearchEventProducer, repositoryRepository)
	adminHandler := web.NewAdminHandler(projectAdminService)
	interactiveEventProducer, err := event.NewInteractiveEventProducer(q)
	if err != nil {
		return nil, err
	}
	serviceService := service.NewService(repositoryRepository, interactiveEventProducer)
	service2 := permModule.Svc
	service3 := intrModule.Svc
	handler := web.NewHandler(serviceService, service2, service3)
	module := &Module{
		AdminHdl: adminHandler,
		Hdl:      handler,
	}
	return module, nil
}

// wire.go:

var (
	adminDAO     dao.ProjectAdminDAO
	adminDAOOnce sync.Once
)

func initAdminDAO(db *egorm.Component) dao.ProjectAdminDAO {
	adminDAOOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
		adminDAO = dao.NewGORMProjectAdminDAO(db)
	})
	return adminDAO
}

func initSyncToSearchEventProducer(q mq.MQ) mq.Producer {
	res, err := q.Producer(event.SyncTopic)
	if err != nil {
		panic(err)
	}
	return res
}
