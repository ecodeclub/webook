// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package ioc

import (
	"github.com/ecodeclub/webook/internal/user"
	"github.com/google/wire"
)

// Injectors from wire.go:

func InitApp() *App {
	db := InitDB()
	cmdable := InitRedis()
	cache := InitCache(cmdable)
	handler := user.InitHandler(db, cache)
	component := initGinxServer(handler)
	provider := InitSession(cmdable)
	app := &App{
		Web: component,
		Sp:  provider,
	}
	return app
}

// wire.go:

var BaseSet = wire.NewSet(InitDB, InitCache, InitRedis)
