package cache

import (
	"context"
	"time"

	"github.com/ecodeclub/ecache"
)

type CaseCache interface {
	// 缓存总数
	GetTotal(ctx context.Context) (int64, error)
	SetTotal(ctx context.Context, total int64) error
}

type caseCache struct {
	ec ecache.Cache
}

func NewCaseCache(ec ecache.Cache) CaseCache {
	return &caseCache{
		ec: &ecache.NamespaceCache{
			C:         ec,
			Namespace: "cases",
		},
	}
}

func (c *caseCache) GetTotal(ctx context.Context) (int64, error) {
	return c.ec.Get(ctx, c.totalKey()).AsInt64()
}

func (c *caseCache) SetTotal(ctx context.Context, total int64) error {
	return c.ec.Set(ctx, c.totalKey(), total, time.Minute*30)
}

func (c *caseCache) totalKey() string {
	return "total"
}
