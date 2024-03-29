package ioc

import (
	"net/http"
	"strings"

	"github.com/ecodeclub/webook-private/nonsense"

	"github.com/ecodeclub/webook/internal/cases"

	"github.com/ecodeclub/webook/internal/label"

	"github.com/ecodeclub/webook/internal/cos"

	baguwen "github.com/ecodeclub/webook/internal/question"

	"github.com/gin-gonic/gin"

	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/gin-contrib/cors"
	"github.com/gotomicro/ego/server/egin"
)

func initGinxServer(sp session.Provider,
	qh *baguwen.Handler,
	qsh *baguwen.QuestionSetHandler,
	lhdl *label.Handler,
	user *user.Handler,
	cosHdl *cos.Handler,
	caseHdl *cases.Handler,
) *egin.Component {
	session.SetDefaultProvider(sp)
	res := egin.Load("web").Build()
	res.Use(cors.New(cors.Config{
		ExposeHeaders:    []string{"X-Refresh-Token", "X-Access-Token"},
		AllowCredentials: true,
		AllowHeaders:     []string{"X-Timestamp", "Authorization", "Content-Type"},
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
	// 虽然叫做 NonSense，但是我还是得告诉你，这是一个安全校验机制
	// 但是我并不能在开源里面放出来，因为知道了如何校验，就知道了如何破解
	// 虽然理论上可以用 plugin 机制，但是 plugin 机制比较容易遇到不兼容的问题
	// 实在不想处理
	res.Use(nonsense.NonSenseV1)
	user.PublicRoutes(res.Engine)
	qh.PublicRoutes(res.Engine)
	cosHdl.PublicRoutes(res.Engine)
	caseHdl.PublicRoutes(res.Engine)
	// 登录校验
	res.Use(session.CheckLoginMiddleware())
	user.PrivateRoutes(res.Engine)
	lhdl.PrivateRoutes(res.Engine)
	qh.PrivateRoutes(res.Engine)
	qsh.PrivateRoutes(res.Engine)
	cosHdl.PrivateRoutes(res.Engine)
	caseHdl.PrivateRoutes(res.Engine)
	return res
}
