//go:build wireinject

package bff

import (
	"github.com/ecodeclub/webook/internal/bff/internal/web"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/interactive"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/google/wire"
)

func InitHandler(intrSvc interactive.Service,
	caseSvc cases.Service,
	queSvc baguwen.Service,
	queSetSvc baguwen.QuestionSetService,
	examSvc baguwen.ExamService) (*web.Handler, error) {
	wire.Build(
		web.NewHandler,
	)
	return new(web.Handler), nil
}

type Hdl web.Handler
