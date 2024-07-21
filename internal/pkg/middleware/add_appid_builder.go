package middleware

import (
	"context"
	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"net/http"
	"strconv"
)

type AddAppIdBuilder struct {
}
const (
	appCtxKey = "app"
)

func NewAddAppIdBuilder()*AddAppIdBuilder {
	return &AddAppIdBuilder{}
}
func (a *AddAppIdBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		gctx := &ginx.Context{Context: ctx}
		appid := ctx.GetHeader(appCtxKey)
		if appid != "" {
			c := ctx.Request.Context()
			app,err := strconv.Atoi(appid)
			if err != nil {
				gctx.AbortWithStatus(http.StatusBadRequest)
				elog.Error("appid设置失败", elog.FieldErr(err))
				return
			}
			newCtx := context.WithValue(c,appCtxKey,uint(app))
			ctx.Request = ctx.Request.WithContext(newCtx)
		}
	}
}


