package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository"
)

var (
	ErrUserDuplicate = repository.ErrUserDuplicate
)

type UserAndService interface {
	Signup(ctx context.Context, u *domain.User) error
}

type UserService struct {
	r repository.UserRepository
}

func NewUserService(r repository.UserRepository) UserAndService {
	return &UserService{
		r: r,
	}
}

func (svc *UserService) Signup(ctx context.Context, u *domain.User) error {
	//hashPwd, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	//if err != nil {
	//	return err
	//}
	//u.Password = string(hashPwd)
	return svc.r.Create(ctx, u)
}
