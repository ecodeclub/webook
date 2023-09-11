package repository

import (
	"context"
	"database/sql"
	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository/dao"
)

type EamilRepository interface {
	Update(ctx context.Context, u domain.User) error
	FindByEmail(ctx context.Context, email string) (dao.User, error)
}

type UserEmailRepository struct {
	dao dao.UserDAO
}

func NewUserEmailRepository(dao dao.UserDAO) EamilRepository {
	return &UserEmailRepository{
		dao: dao,
	}
}

func (ur *UserEmailRepository) Update(ctx context.Context, u domain.User) error {
	return ur.dao.Update(ctx, dao.User{
		Id:       u.Id,
		Email:    u.Email,
		Password: u.Password,
		EmailVerify: sql.NullByte{
			Byte:  u.EmailVerify,
			Valid: true,
		},
		UpdateTime: u.UpdateTime.UnixMilli(),
	})
}

func (ur *UserEmailRepository) FindByEmail(ctx context.Context, email string) (dao.User, error) {
	user, err := ur.dao.FindByEmail(ctx, email)
	if err != nil {
		return dao.User{}, err
	}
	return user, err
}
