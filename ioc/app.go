package ioc

import (
	"github.com/ecodeclub/ginx/session"
	"github.com/gotomicro/ego/server/egin"
)

type App struct {
	Web *egin.Component
	Sp  session.Provider
}
