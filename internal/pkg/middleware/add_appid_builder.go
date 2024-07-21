package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/ecodeclub/ginx"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type AddAppIdBuilder struct {
}

type AppContextType string

const (
	AppCtxKey AppContextType = "app"
)

func NewAddAppIdBuilder() *AddAppIdBuilder {
	return &AddAppIdBuilder{}
}
func (a *AddAppIdBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		gctx := &ginx.Context{Context: ctx}
		appid := ctx.GetHeader(string(AppCtxKey))
		if appid != "" {
			c := ctx.Request.Context()
			app, err := strconv.Atoi(appid)
			if err != nil {
				gctx.AbortWithStatus(http.StatusBadRequest)
				elog.Error("appid设置失败", elog.FieldErr(err))
				return
			}
			newCtx := CtxWithAppId(c, uint(app))
			ctx.Request = ctx.Request.WithContext(newCtx)
		}
	}
}

func AppID(ctx context.Context) (uint, bool) {
	app := ctx.Value(AppCtxKey)
	if app == nil {
		return 0, false
	}
	v, ok := app.(uint)
	return v, ok
}

func CtxWithAppId(ctx context.Context, appid uint) context.Context {
	return context.WithValue(ctx, AppCtxKey, appid)
}
