package service

import (
	"context"
	"errors"

	"github.com/lithammer/shortuuid/v4"

	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/ecodeclub/webook/internal/user/internal/repository"
	"github.com/gotomicro/ego/core/elog"
)

//go:generate mockgen -source=./user.go -package=svcmocks -destination=mocks/user.mock.go UserService
type UserService interface {
	Profile(ctx context.Context, id int64) (domain.User, error)
	// FindOrCreateByWechat 查找或者初始化
	// 随着业务增长，这边可以考虑拆分出去作为一个新的 Service
	FindOrCreateByWechat(ctx context.Context, info domain.WechatInfo) (domain.User, bool, error)

	// UpdateNonSensitiveInfo 更新非敏感数据
	// 你可以在这里进一步补充究竟哪些数据会被更新
	UpdateNonSensitiveInfo(ctx context.Context, user domain.User) error
}

type userService struct {
	repo   repository.UserRepository
	logger *elog.Component
}

func NewUserService(repo repository.UserRepository) UserService {
	return &userService{
		repo:   repo,
		logger: elog.DefaultLogger,
	}
}

func (svc *userService) UpdateNonSensitiveInfo(ctx context.Context, user domain.User) error {
	// 不让修改序列号
	user.SN = ""
	return svc.repo.Update(ctx, user)
}

func (svc *userService) FindOrCreateByWechat(ctx context.Context,
	info domain.WechatInfo) (domain.User, bool, error) {
	// 类似于手机号的过程，大部分人只是扫码登录，也就是数据在我们这里是有的
	u, err := svc.repo.FindByWechat(ctx, info.OpenId)
	if !errors.Is(err, repository.ErrUserNotFound) {
		return u, false, err
	}
	sn := shortuuid.New()
	id, err := svc.repo.Create(ctx, domain.User{
		WechatInfo: info,
		SN:         sn,
		Nickname:   sn[:4],
	})
	return domain.User{
		Id:         id,
		WechatInfo: info,
	}, true, err
}

func (svc *userService) Profile(ctx context.Context,
	id int64) (domain.User, error) {
	// 在系统内部，基本上都是用 ID 的。
	// 有些人的系统比较复杂，有一个 GUID（global unique ID）
	return svc.repo.FindById(ctx, id)
}
