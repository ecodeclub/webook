package repository

import (
	"context"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ekit/sqlx"

	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/ecodeclub/webook/internal/user/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/user/internal/repository/dao"
)

//go:generate mockgen -source=./user.go -package=repomocks -destination=mocks/user.mock.go UserRepository
type UserRepository interface {
	Create(ctx context.Context, u domain.User) (int64, error)
	// Update 更新数据，只有非 0 值才会更新
	Update(ctx context.Context, u domain.User) error
	// FindByWechat 按照 unionId 来查询
	FindByWechat(ctx context.Context, unionId string) (domain.User, error)
	FindById(ctx context.Context, id int64) (domain.User, error)
	FindByIds(ctx context.Context, ids []int64) ([]domain.User, error)
	FindByPhone(ctx context.Context, phone string) (domain.User, error)
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
	return ur.dao.Insert(ctx, ur.domainToEntity(u))
}

func (ur *CachedUserRepository) FindByWechat(ctx context.Context,
	unionId string) (domain.User, error) {
	u, err := ur.dao.FindByWechat(ctx, unionId)
	return ur.entityToDomain(u), err
}
func (ur *CachedUserRepository) FindByPhone(ctx context.Context, phone string) (domain.User, error) {
	u, err := ur.dao.FindByPhone(ctx, phone)
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

// FindByIds todo 从数据库中取
func (ur *CachedUserRepository) FindByIds(ctx context.Context, ids []int64) ([]domain.User, error) {
	if len(ids) == 0 {
		return []domain.User{}, nil
	}
	us, err := ur.dao.FindByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	return slice.Map(us, func(idx int, src dao.User) domain.User {
		return ur.entityToDomain(src)
	}), nil
}

func (ur *CachedUserRepository) domainToEntity(u domain.User) dao.User {
	return dao.User{
		Id:               u.Id,
		SN:               u.SN,
		Nickname:         u.Nickname,
		Avatar:           u.Avatar,
		Phone:            sqlx.NewNullString(u.Phone),
		WechatUnionId:    sqlx.NewNullString(u.WechatInfo.UnionId),
		WechatOpenId:     sqlx.NewNullString(u.WechatInfo.OpenId),
		WechatMiniOpenId: sqlx.NewNullString(u.WechatInfo.MiniOpenId),
	}
}

func (ur *CachedUserRepository) entityToDomain(ue dao.User) domain.User {
	return domain.User{
		Id:       ue.Id,
		Nickname: ue.Nickname,
		SN:       ue.SN,
		Avatar:   ue.Avatar,
		Phone:    ue.Phone.String,
		WechatInfo: domain.WechatInfo{
			OpenId:     ue.WechatOpenId.String,
			UnionId:    ue.WechatUnionId.String,
			MiniOpenId: ue.WechatMiniOpenId.String,
		},
	}
}
