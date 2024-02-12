package ioc

import (
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/gotomicro/ego/server/egin"
)

func initGinxServer(user *user.Handler) *egin.Component {
	res := egin.Load("web").Build()
	user.PublicRoutes(res.Engine)
	// 登录校验
	res.Use(session.CheckLoginMiddleware())
	user.PrivateRoutes(res.Engine)
	return res
}
