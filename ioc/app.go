package ioc

import (
	"context"

	"github.com/gotomicro/ego/server/egin"
	"github.com/gotomicro/ego/task/ecron"
	"github.com/gotomicro/ego/task/ejob"
)

type App struct {
	Web       *egin.Component
	Admin     AdminServer
	Crons     []ecron.Ecron
	Jobs      []ejob.Ejob
	Consumers []Consumer
}

type Consumer interface {
	Start(ctx context.Context)
}
