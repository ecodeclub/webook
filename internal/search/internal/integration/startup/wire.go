//go:build wireinject

package startup

import (
	baguwen "github.com/ecodeclub/webook/internal/search"
	"github.com/ecodeclub/webook/internal/search/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitHandler() (*web.Handler, error) {
	wire.Build(testioc.BaseSet, baguwen.InitModule,
		wire.FieldsOf(new(*baguwen.Module), "Hdl"))
	return new(web.Handler), nil
}
