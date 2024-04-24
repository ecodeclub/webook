package ioc

import (
	"github.com/gotomicro/ego/server/egin"
	"github.com/gotomicro/ego/task/ecron"
)

type App struct {
	Web   *egin.Component
	Admin AdminServer
	Jobs  []*ecron.Component
}
