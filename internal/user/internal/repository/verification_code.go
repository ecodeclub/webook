package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/user/internal/repository/cache"
)

type VerificationCodeRepo interface {
	SetPhoneCode(ctx context.Context, phone string, code string) error
	GetPhoneCode(ctx context.Context, phone string) (string, error)
}
type verificationRepository struct {
	cache.VerificationCodeCache
}

func NewVerificationCodeRepository(smsCache cache.VerificationCodeCache) VerificationCodeRepo {
	return &verificationRepository{
		VerificationCodeCache: smsCache,
	}
}
