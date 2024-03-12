//go:build wireinject

package startup

import (
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/ecodeclub/webook/internal/user/internal/repository"
	"github.com/ecodeclub/webook/internal/user/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/user/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/ecodeclub/webook/internal/user/internal/web"
	"github.com/google/wire"
)

func InitHandler(weSvc service.OAuth2Service, creators []string) *user.Handler {
	wire.Build(web.NewHandler,
		testioc.BaseSet,
		service.NewUserService,
		dao.NewGORMUserDAO,
		cache.NewUserECache,
		repository.NewCachedUserRepository)
	return new(user.Handler)
}
