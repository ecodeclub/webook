//go:build wireinject

package ioc

import (
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/google/wire"
)

var BaseSet = wire.NewSet(InitDB, InitCache, InitRedis)

func InitApp() (*App, error) {
	wire.Build(wire.Struct(new(App), "*"),
		BaseSet,
		baguwen.InitHandler,
		user.InitHandler,
		InitSession,
		initGinxServer)
	return new(App), nil
}
