package repository

import (
	"context"
	"database/sql"

	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/ecodeclub/webook/internal/user/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/user/internal/repository/dao"
)

//go:generate mockgen -source=./user.go -package=repomocks -destination=mocks/user.mock.go UserRepository
type UserRepository interface {
	Create(ctx context.Context, u domain.User) (int64, error)
	// Update 更新数据，只有非 0 值才会更新
	Update(ctx context.Context, u domain.User) error
	// FindByWechat 暂时可以认为按照 openId来查询
	// 将来可能需要按照 unionId 来查询
	FindByWechat(ctx context.Context, openId string) (domain.User, error)
	FindById(ctx context.Context, id int64) (domain.User, error)
}

// CachedUserRepository 使用了缓存的 repository 实现
type CachedUserRepository struct {
	dao   dao.UserDAO
	cache cache.UserCache
}

// NewCachedUserRepository 支持缓存的实现
func NewCachedUserRepository(d dao.UserDAO,
	c cache.UserCache) UserRepository {
	return &CachedUserRepository{
		dao:   d,
		cache: c,
	}
}

func (ur *CachedUserRepository) Update(ctx context.Context, u domain.User) error {
	err := ur.dao.UpdateNonZeroFields(ctx, ur.domainToEntity(u))
	if err != nil {
		return err
	}
	return ur.cache.Delete(ctx, u.Id)
}

func (ur *CachedUserRepository) Create(ctx context.Context, u domain.User) (int64, error) {
	return ur.dao.Insert(ctx, dao.User{
		SN:       u.SN,
		Nickname: u.Nickname,
		WechatUnionId: sql.NullString{
			String: u.WechatInfo.UnionId,
			Valid:  u.WechatInfo.UnionId != "",
		},
		WechatOpenId: sql.NullString{
			String: u.WechatInfo.OpenId,
			Valid:  u.WechatInfo.OpenId != "",
		},
	})
}

func (ur *CachedUserRepository) FindByWechat(ctx context.Context,
	openId string) (domain.User, error) {
	u, err := ur.dao.FindByWechat(ctx, openId)
	return ur.entityToDomain(u), err
}

func (ur *CachedUserRepository) FindById(ctx context.Context,
	id int64) (domain.User, error) {
	u, err := ur.cache.Get(ctx, id)
	if err == nil {
		return u, err
	}
	ue, err := ur.dao.FindById(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	u = ur.entityToDomain(ue)
	// 忽略掉这里的错误
	_ = ur.cache.Set(ctx, u)
	return u, nil
}

func (ur *CachedUserRepository) domainToEntity(u domain.User) dao.User {
	return dao.User{
		Id:       u.Id,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
	}
}

func (ur *CachedUserRepository) entityToDomain(ue dao.User) domain.User {
	return domain.User{
		Id:       ue.Id,
		Nickname: ue.Nickname,
		SN:       ue.SN,
		Avatar:   ue.Avatar,
		WechatInfo: domain.WechatInfo{
			OpenId:  ue.WechatOpenId.String,
			UnionId: ue.WechatUnionId.String,
		},
	}
}
