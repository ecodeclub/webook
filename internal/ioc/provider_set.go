package ioc

import (
	"github.com/ecodeclub/webook/internal/repository"
	"github.com/ecodeclub/webook/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/service"
	"github.com/ecodeclub/webook/internal/web"
	"github.com/google/wire"
)

var (
	UserProviders = wire.NewSet(web.NewUserHandler, service.NewUserService, repository.NewUserInfoRepository, dao.NewUserInfoDAO)
)
