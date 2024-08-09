//go:build wireinject

package bff

import (
	"github.com/ecodeclub/webook/internal/bff/internal/web"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/interactive"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/google/wire"
)

func InitModule(intrModule *interactive.Module,
	caseModule *cases.Module,
	queModule *baguwen.Module) (*Module, error) {
	wire.Build(
		web.NewHandler,
		wire.FieldsOf(new(*baguwen.Module), "Svc", "SetSvc", "ExamSvc"),
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*cases.Module), "SetSvc", "Svc"),
		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
}

type Handler = web.Handler
