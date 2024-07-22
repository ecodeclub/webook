package middleware

import (
	"net/http"
	"strconv"

	"github.com/ecodeclub/webook/internal/pkg/ectx"

	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type CheckAppIdBuilder struct {
}

const (
	appIDHeader = "app"
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
		c := ctx.Request.Context()
		app, err := strconv.Atoi(appid)
		if err != nil {
			gctx.AbortWithStatus(http.StatusBadRequest)
			elog.Error("appid设置失败", elog.FieldErr(err))
			return
		}
		newCtx := ectx.CtxWithAppId(c, uint(app))
		ctx.Request = ctx.Request.WithContext(newCtx)
	}
}
