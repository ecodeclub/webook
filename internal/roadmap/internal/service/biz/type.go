package biz

import (
	"context"

	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
)

// Service 作为一个聚合服务，下沉到这里以减轻 web 的逻辑负担
type Service interface {
	// GetBizs bizs 和 ids 的长度必须一样
	// 返回值是 biz-id-Biz 的结构
	GetBizs(ctx context.Context, bizs []string, ids []int64) (map[string]map[int64]domain.Biz, error)
}

type Strategy interface {
	GetBizsByIds(ctx context.Context, ids []int64) (map[int64]domain.Biz, error)
}
