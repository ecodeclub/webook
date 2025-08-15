package cache

import (
	"context"
	"errors"
	"time"

	"github.com/ecodeclub/ecache"
)

var ErrKeyNotFound = errors.New("key not found")

type VerificationCodeCache interface {
	SetPhoneCode(ctx context.Context, phone string, code string) error
	GetPhoneCode(ctx context.Context, phone string) (string, error)
}

type verificationCodeCache struct {
	cache ecache.Cache
	// 过期时间
	expiration time.Duration
}

// NewVerificationCodeCache 注意缓存前缀
func NewVerificationCodeCache(c ecache.Cache) VerificationCodeCache {
	return &verificationCodeCache{
		cache: &ecache.NamespaceCache{
			Namespace: "sms:",
			C:         c,
		},
		// 默认五分钟
		expiration: time.Minute * 5,
	}
}

func (s *verificationCodeCache) SetPhoneCode(ctx context.Context, phone string, code string) error {
	return s.cache.Set(ctx, phone, code, s.expiration)
}

func (s *verificationCodeCache) GetPhoneCode(ctx context.Context, phone string) (string, error) {
	val := s.cache.Get(ctx, phone)
	if val.Err != nil {
		return "", val.Err
	}
	if val.KeyNotFound() {
		return "", ErrKeyNotFound
	}
	return val.Val.(string), nil
}
