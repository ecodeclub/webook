//go:build wireinject

package startup

import (
	"github.com/ecodeclub/webook/internal/cases"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/ecodeclub/webook/internal/skill"
	"github.com/ecodeclub/webook/internal/skill/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitHandler(bm *baguwen.Module, cm *cases.Module) (*web.Handler, error) {
	wire.Build(testioc.BaseSet, skill.InitHandler)
	return new(web.Handler), nil
}
