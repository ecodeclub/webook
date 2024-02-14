//go:build wireinject

package user

import (
	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/webook/internal/user/internal/repository"
	"github.com/ecodeclub/webook/internal/user/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/user/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/ecodeclub/webook/internal/user/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"github.com/gotomicro/ego/core/econf"
)

var ProviderSet = wire.NewSet(web.NewHandler,
	cache.NewUserECache,
	InitDAO,
	InitWechatService,
	service.NewUserService,
	repository.NewCachedUserRepository)

func InitHandler(db *egorm.Component, cache ecache.Cache) *Handler {
	wire.Build(ProviderSet)
	return new(Handler)
}

func InitWechatService() service.OAuth2Service {
	type Config struct {
		AppSecretID  string `yaml:"appSecretID"`
		AppSecretKey string `yaml:"appSecretKey"`
	}
	var cfg Config
	err := econf.UnmarshalKey("wechat", &cfg)
	if err != nil {
		panic(err)
	}
	return service.NewWechatService(cfg.AppSecretID, cfg.AppSecretKey)
}

func InitDAO(db *egorm.Component) dao.UserDAO {
	err := dao.InitTables(db)
	if err != nil {
		panic(err)
	}
	return dao.NewGORMUserDAO(db)
}

// Handler 暴露出去给 ioc 使用
type Handler = web.Handler
