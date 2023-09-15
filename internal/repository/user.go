package repository

import (
	"context"
	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository/dao"
)

var (
	ErrUserDuplicate = dao.ErrUserDuplicate
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

func (ur *UserInfoRepository) userToDomain(u dao.User) domain.User {
	return domain.User{
		Id:            u.Id,
		EmailVerified: u.EmailVerified,
		Email:         u.Email,
		Password:      u.Password,
	}
}

func (ur *UserInfoRepository) Create(ctx context.Context, u *domain.User) error {
	return ur.dao.Insert(ctx, dao.User{
		Id:            u.Id,
		Email:         u.Email,
		Password:      u.Password,
		CreateTime:    u.CreateTime.UnixMilli(),
		UpdateTime:    u.UpdateTime.UnixMilli(),
		EmailVerified: false,
	})
}
func (ur *UserInfoRepository) UpdateEmailVerified(ctx context.Context, email string) error {
	return ur.dao.UpdateEmailVerified(ctx, email)
}

func (ur *UserInfoRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := ur.dao.FindByEmail(ctx, email)
	if err != nil {
		return domain.User{}, err
	}
	return ur.userToDomain(user), err
}
