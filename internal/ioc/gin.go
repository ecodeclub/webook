package ioc

import (
	"github.com/ecodeclub/webook/internal/web"
	"github.com/gin-gonic/gin"
)

func InitWebServer(funcs []gin.HandlerFunc, userHdl *web.UserHandler) *gin.Engine {
	server := gin.Default()
	server.Use(funcs...)
	userHdl.RegisterRoutes(server)
	return server
}

func GinMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{}
}
