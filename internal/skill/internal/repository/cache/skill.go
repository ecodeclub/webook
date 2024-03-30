package cache

import (
	"context"
	"time"

	"github.com/ecodeclub/ecache"
)

type SkillCache interface {
	// 缓存总数
	GetTotal(ctx context.Context) (int64, error)
	SetTotal(ctx context.Context, total int64) error
}

type skillCache struct {
	ec ecache.Cache
}

func NewSkillCache(ec ecache.Cache) SkillCache {
	return &skillCache{
		ec: &ecache.NamespaceCache{
			C:         ec,
			Namespace: "skill",
		},
	}
}

func (s *skillCache) GetTotal(ctx context.Context) (int64, error) {
	return s.ec.Get(ctx, s.totalKey()).AsInt64()
}

func (s *skillCache) SetTotal(ctx context.Context, total int64) error {
	return s.ec.Set(ctx, s.totalKey(), total, time.Minute*30)
}

func (s *skillCache) totalKey() string {
	return "total"
}
