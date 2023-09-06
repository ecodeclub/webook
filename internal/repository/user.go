package repository

import (
	"context"
	"webook/internal/domain"
	"webook/internal/repository/dao"
)

var (
	ErrUserDuplicate = dao.ErrUserDuplicate
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
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
