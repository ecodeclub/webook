//go:build wireinject

package startup

import (
	"github.com/ecodeclub/webook/internal/company"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitModule() (*company.Module, error) {
	wire.Build(testioc.BaseSet, company.InitModule)
	return new(company.Module), nil
}
