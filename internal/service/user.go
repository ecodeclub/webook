package service

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository"
)

var (
	ErrUserDuplicate     = repository.ErrUserDuplicate
	ErrUserEmailVerified = errors.New("邮箱已激活")
)

type UserService interface {
	Signup(ctx context.Context, u *domain.User) error
	EmailVerify(ctx context.Context, email string) error
}

type userService struct {
	repo   repository.UserRepository
	logger *zap.Logger
}

func NewUserService(repo repository.UserRepository, logger *zap.Logger) UserService {
	return &userService{
		repo:   repo,
		logger: logger,
	}
}

func (svc *userService) Signup(ctx context.Context, u *domain.User) error {
	// hashPwd, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	// if err != nil {
	//	return err
	// }
	// u.Password = string(hashPwd)
	return svc.repo.Create(ctx, u)
}

func (svc *userService) EmailVerify(ctx context.Context, email string) error {
	// 这里可能有并发问题，但概率微乎其微，且可以不进行处理，因为多更新一次并不影响。
	u, err := svc.repo.FindByEmail(ctx, email)
	if err != nil {
		return err
	}
	if u.EmailVerified {
		return ErrUserEmailVerified
	}

	err = svc.repo.UpdateEmailVerified(ctx, email)
	if err != nil {
		svc.logger.Error("更新邮箱验证失败", zap.Error(err))
		return err
	}
	return nil
}
