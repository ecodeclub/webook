//go:build wireinject

package company

import (
	"sync"

	"github.com/ecodeclub/webook/internal/company/internal/repository"
	"github.com/ecodeclub/webook/internal/company/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/company/internal/service"
	"github.com/ecodeclub/webook/internal/company/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

var HandlerSet = wire.NewSet(
	InitService,
	web.NewCompanyHandler,
)

func InitModule(db *egorm.Component) (*Module, error) {
	wire.Build(HandlerSet, wire.Struct(new(Module), "*"))
	return new(Module), nil
}

func InitService(db *egorm.Component) Service {
	wire.Build(
		InitTablesOnce,
		repository.NewCompanyRepository,
		service.NewCompanyService,
	)
	return nil
}

var once = &sync.Once{}

func InitTablesOnce(db *egorm.Component) dao.CompanyDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
	})
	return dao.NewGORMCompanyDAO(db)
}
