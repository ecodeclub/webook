package middleware

import (
	"net/http"
	"strconv"

	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type CheckAppIdBuilder struct {
}

const (
	appIDHeader = "app"
	AppCtxKey   = "app"
)

func NewCheckAppIdBuilder() *CheckAppIdBuilder {
	return &CheckAppIdBuilder{}
}
func (a *CheckAppIdBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		gctx := &ginx.Context{Context: ctx}
		appid := ctx.GetHeader(appIDHeader)
		if appid == "" {
			return
		}
		app, err := strconv.Atoi(appid)
		if err != nil {
			gctx.AbortWithStatus(http.StatusBadRequest)
			elog.Error("appid设置失败", elog.FieldErr(err))
			return
		}
		ctx.Set(AppCtxKey, uint(app))
	}
}
