package ioc

import (
	"context"

	"github.com/gotomicro/ego/server/egin"
	"github.com/gotomicro/ego/task/ecron"
)

type App struct {
	Web       *egin.Component
	Admin     AdminServer
	Crons     []ecron.Ecron
	Consumers []Consumer
}

type Consumer interface {
	Start(ctx context.Context)
}
