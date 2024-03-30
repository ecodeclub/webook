//go:build wireinject

package startup

import (
	baguwen "github.com/ecodeclub/webook/internal/skill"
	"github.com/ecodeclub/webook/internal/skill/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitHandler() (*web.Handler, error) {
	wire.Build(testioc.BaseSet, baguwen.InitHandler)
	return new(web.Handler), nil
}
