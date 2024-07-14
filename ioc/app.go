package ioc

import (
	"github.com/gotomicro/ego/server/egin"
	"github.com/gotomicro/ego/task/ecron"
	"github.com/gotomicro/ego/task/ejob"
)

type App struct {
	Web   *egin.Component
	Admin AdminServer
	Crons []ecron.Ecron
	Jobs  []ejob.Ejob
}
