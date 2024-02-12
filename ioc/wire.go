//go:build wireinject

package ioc

import (
	"github.com/ecodeclub/webook/internal/user"
	"github.com/google/wire"
)

var BaseSet = wire.NewSet(InitDB, InitCache, InitRedis)

func InitApp() *App {
	wire.Build(wire.Struct(new(App), "*"),
		BaseSet,
		user.InitHandler,
		InitSession,
		initGinxServer)
	return new(App)
}
