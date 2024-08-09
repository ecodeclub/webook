//go:build wireinject

package bff

import (
	"github.com/ecodeclub/webook/internal/bff/internal/web"
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/interactive"
	baguwen "github.com/ecodeclub/webook/internal/question"
	"github.com/google/wire"
)

func InitHandler(intrModule *interactive.Module,
	caseModule *cases.Module,
	queSvc *baguwen.Module) (*web.Handler, error) {
	wire.Build(
		web.NewHandler,
		wire.FieldsOf(new(*interactive.Module), "Svc"),
		wire.FieldsOf(new(*baguwen.Module), "Svc", "SetSvc", "ExamSvc"),
		wire.FieldsOf(new(*cases.Module), "Svc", "SetSvc"),
	)
	return new(web.Handler), nil
}

type Hdl web.Handler
