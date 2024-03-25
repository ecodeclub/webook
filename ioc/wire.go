//go:build wireinject

package ioc

import (
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/cos"
	"github.com/ecodeclub/webook/internal/label"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/google/wire"
)

var BaseSet = wire.NewSet(InitDB, InitCache, InitRedis, InitCosConfig)

func InitApp() (*App, error) {
	wire.Build(wire.Struct(new(App), "*"),
		BaseSet,
		cos.InitHandler,
		baguwen.InitHandler,
		baguwen.InitQuestionSetHandler,
		InitUserHandler,
		InitSession,
		label.InitHandler,
		cases.InitHandler,
		initGinxServer)
	return new(App), nil
}
