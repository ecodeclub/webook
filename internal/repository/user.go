package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository/dao"
)

var (
	ErrUserDuplicate = dao.ErrUserDuplicate
	ErrUserNotFound  = dao.ErrDataNotFound
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	UpdateEmailVerified(ctx context.Context, email string) error
	FindByEmail(ctx context.Context, email string) (domain.User, error)
}

type UserInfoRepository struct {
	dao dao.UserDAO
}

func NewUserInfoRepository(dao dao.UserDAO) UserRepository {
	return &UserInfoRepository{
		dao: dao,
	}
}

func (ur *UserInfoRepository) Create(ctx context.Context, u *domain.User) error {
	return ur.dao.Insert(ctx, dao.User{
		Id:         u.Id,
		Email:      u.Email,
		Password:   u.Password,
		CreateTime: u.CreateTime.UnixMilli(),
		UpdateTime: u.UpdateTime.UnixMilli(),
	})
}

func (ur *UserInfoRepository) UpdateEmailVerified(
	ctx context.Context, email string) error {
	return ur.dao.UpdateEmailVerifiedByEmail(ctx, email)
}

func (ur *UserInfoRepository) FindByEmail(ctx context.Context,
	email string) (domain.User, error) {
	u, err := ur.dao.FindByEmail(ctx, email)
	return ur.entityToDomain(u), err
}

func (ur *UserInfoRepository) entityToDomain(ue dao.User) domain.User {
	ctime := time.Time{}
	if ue.CreateTime != 0 {
		ctime = time.UnixMilli(ue.CreateTime)
	}
	utime := time.Time{}
	if ue.CreateTime != 0 {
		utime = time.UnixMilli(ue.UpdateTime)
	}
	return domain.User{
		Id:            ue.Id,
		Email:         ue.Email,
		Password:      ue.Password,
		EmailVerified: ue.EmailVerified,
		CreateTime:    ctime,
		UpdateTime:    utime,
	}
}

func (ur *UserInfoRepository) domainToEntity(u domain.User) dao.User {
	return dao.User{
		Id:            u.Id,
		Email:         u.Email,
		Password:      u.Password,
		EmailVerified: u.EmailVerified,
		CreateTime:    u.CreateTime.UnixMilli(),
		UpdateTime:    u.UpdateTime.UnixMilli(),
	}
}
