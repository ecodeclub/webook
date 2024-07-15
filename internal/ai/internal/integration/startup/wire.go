//go:build wireinject

package startup

import (
	"github.com/ecodeclub/webook/internal/ai"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/service"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt/sdk"
	"github.com/ecodeclub/webook/internal/credit"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitModule(
	aisdk sdk.GPTSdk, creditSvc credit.Service) (*ai.Module, error) {
	wire.Build(
		testioc.InitDB,
		ai.InitGPTDAO,
		repository.NewGPTLogRepo,
		ai.InitHandlers,
		service.NewGPTService,
		wire.Struct(new(ai.Module), "*"),
	)
	return new(ai.Module), nil
}
