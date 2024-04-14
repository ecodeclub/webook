// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package product

import (
	"sync"

	"github.com/ecodeclub/webook/internal/product/internal/domain"
	"github.com/ecodeclub/webook/internal/product/internal/repository"
	"github.com/ecodeclub/webook/internal/product/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/product/internal/service"
	"github.com/ecodeclub/webook/internal/product/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

// Injectors from wire.go:

func InitHandler(db *gorm.DB) *web.Handler {
	service := InitService(db)
	handler := web.NewHandler(service)
	return handler
}

func InitService(db *gorm.DB) service.Service {
	productDAO := InitTablesOnce(db)
	productRepository := repository.NewProductRepository(productDAO)
	serviceService := service.NewService(productRepository)
	return serviceService
}

// wire.go:

var ServiceSet = wire.NewSet(
	InitTablesOnce, repository.NewProductRepository, service.NewService,
)

var HandlerSet = wire.NewSet(
	InitService, web.NewHandler,
)

var once = &sync.Once{}

func InitTablesOnce(db *egorm.Component) dao.ProductDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
	})
	return dao.NewProductGORMDAO(db)
}

type Handler = web.Handler

type Service = service.Service

type Product = domain.Product

type SKU = domain.SKU

type SPU = domain.SPU

type Status = domain.Status

const StatusOffShelf = domain.StatusOffShelf
const StatusOnShelf = domain.StatusOnShelf

const SaleTypeUnlimited = domain.SaleTypeUnlimited
