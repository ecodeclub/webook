package ioc

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/gin-contrib/cors"
	"github.com/gotomicro/ego/server/egin"
)

func initGinxServer(sp session.Provider, user *user.Handler) *egin.Component {
	session.SetDefaultProvider(sp)
	res := egin.Load("web").Build()
	res.Use(cors.New(cors.Config{
		ExposeHeaders:    []string{"x-refresh-token", "x-access-token"},
		AllowCredentials: true,
		AllowHeaders:     []string{"Authorization"},
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "http://localhost") {
				return true
			}
			// 只允许我的域名过来的
			return strings.Contains(origin, "meoying.com")
		},
	}))
	res.GET("/hello", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "hello, world!")
	})
	user.PublicRoutes(res.Engine)
	// 登录校验
	res.Use(session.CheckLoginMiddleware())
	user.PrivateRoutes(res.Engine)
	return res
}
