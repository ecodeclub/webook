//go:build wireinject

package ioc

import (
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/cos"
	"github.com/ecodeclub/webook/internal/label"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/skill"
	"github.com/google/wire"
)

var BaseSet = wire.NewSet(InitDB, InitCache, InitRedis, InitCosConfig)

func InitApp() (*App, error) {
	wire.Build(wire.Struct(new(App), "*"),
		BaseSet,
		cos.InitHandler,
		baguwen.InitModule,
		wire.FieldsOf(new(*baguwen.Module), "Hdl", "QsHdl"),
		InitUserHandler,
		InitSession,
		label.InitHandler,
		cases.InitModule,
		wire.FieldsOf(new(*cases.Module), "Hdl"),
		skill.InitHandler,
		initGinxServer)
	return new(App), nil
}
