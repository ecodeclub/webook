//go:build wireinject

package main

import (
	"github.com/ecodeclub/webook/internal/ioc"
	"github.com/ecodeclub/webook/internal/service/email/gomail"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

func InitWebServer() *gin.Engine {
	panic(wire.Build(
		ioc.UserProviders,

		gomail.NewEmailService,

		ioc.InitEmailCfg,
		ioc.InitDB,
		ioc.GinMiddlewares,
		ioc.InitWebServer,
	))
}
