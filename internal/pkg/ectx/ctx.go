package ectx

import "context"

type appContextType string

var (
	appCtxKey appContextType = "app"
)
func GetAppIdFromCtx(ctx context.Context) (uint, bool) {
	app := ctx.Value(appCtxKey)
	if app == nil {
		return 0, false
	}
	v, ok := app.(uint)
	return v, ok
}

func CtxWithAppId(ctx context.Context, appid uint) context.Context {
	return context.WithValue(ctx, appCtxKey, appid)
}