package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/feedback/internal/domain"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository"
)

type Service interface {
	// 管理端
	// 列表 根据交互来
	List(ctx context.Context, feedBack domain.FeedBack, offset, limit int) ([]domain.FeedBack, error)
	PendingCount(ctx context.Context) (int64, error)
	// 详情
	Info(ctx context.Context, id int64) (domain.FeedBack, error)
	// 处理 反馈
	UpdateStatus(ctx context.Context, id int64, status domain.FeedBackStatus) error
	//	c端
	// 添加
	Create(ctx context.Context, feedback domain.FeedBack) error
}

type service struct {
	repo repository.FeedBackRepo
}

func (s *service) PendingCount(ctx context.Context) (int64, error) {
	return s.repo.PendingCount(ctx)
}

func (s *service) Info(ctx context.Context, id int64) (domain.FeedBack, error) {
	return s.repo.Info(ctx, id)
}

func (s *service) UpdateStatus(ctx context.Context, id int64, status domain.FeedBackStatus) error {
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *service) Create(ctx context.Context, feedback domain.FeedBack) error {
	return s.repo.Create(ctx, feedback)
}

func (s *service) List(ctx context.Context, feedBack domain.FeedBack, offset, limit int) ([]domain.FeedBack, error) {
	return s.repo.List(ctx, feedBack, offset, limit)
}

func NewService(repo repository.FeedBackRepo) Service {
	return &service{
		repo: repo,
	}
}
