// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package startup

import (
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/permission"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/ecodeclub/webook/internal/user/internal/event"
	"github.com/ecodeclub/webook/internal/user/internal/repository"
	"github.com/ecodeclub/webook/internal/user/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/user/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/ecodeclub/webook/internal/user/internal/web"
)

// Injectors from wire.go:

func InitHandler(weSvc wechatWebOAuth2Service, weMiniSvc wechatMiniOAuth2Service, mem *member.Module, perm *permission.Module, sp session.Provider, creators []string) *web.Handler {
	db := testioc.InitDB()
	userDAO := dao.NewGORMUserDAO(db)
	ecacheCache := testioc.InitCache()
	userCache := cache.NewUserECache(ecacheCache)
	userRepository := repository.NewCachedUserRepository(userDAO, userCache)
	mq := testioc.InitMQ()
	registrationEventProducer := initRegistrationEventProducer(mq)
	userService := service.NewUserService(userRepository, registrationEventProducer)
	serviceService := mem.Svc
	service2 := perm.Svc
	handler := iniHandler(weSvc, weMiniSvc, userService, serviceService, service2, sp, creators)
	return handler
}

func InitModule() *user.Module {
	db := testioc.InitDB()
	userDAO := dao.NewGORMUserDAO(db)
	ecacheCache := testioc.InitCache()
	userCache := cache.NewUserECache(ecacheCache)
	userRepository := repository.NewCachedUserRepository(userDAO, userCache)
	mq := testioc.InitMQ()
	registrationEventProducer := initRegistrationEventProducer(mq)
	userService := service.NewUserService(userRepository, registrationEventProducer)
	module := &user.Module{
		Svc: userService,
	}
	return module
}

// wire.go:

func iniHandler(
	weSvc wechatWebOAuth2Service,
	weMiniSvc wechatMiniOAuth2Service,
	userSvc service.UserService,
	memberSvc member.Service,
	permissionSvc permission.Service,
	sp session.Provider,
	creators []string) *web.Handler {
	return web.NewHandler(weSvc, weMiniSvc, userSvc, memberSvc, permissionSvc, sp, creators)
}

func initRegistrationEventProducer(q mq.MQ) event.RegistrationEventProducer {
	p, err := event.NewRegistrationEventProducer(q)
	if err != nil {
		panic(err)
	}
	return p
}
