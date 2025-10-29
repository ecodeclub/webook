package biz

import (
	"context"

	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
)

type TourStrategy struct {
}

func NewTourStrategy() Strategy {
	return &TourStrategy{}
}

func (t *TourStrategy) GetBizsByIds(ctx context.Context, ids []int64) (map[int64]domain.Biz, error) {
	// 是在首页展示的路线图的关联业务，以后有需要扩展
	return map[int64]domain.Biz{
		1: domain.Biz{
			Biz:   "tourGuide",
			Title: "旅程指南",
		},
	}, nil
}
