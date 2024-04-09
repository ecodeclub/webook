package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/feedback/internal/domain"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository/dao"
)

type FeedbackRepository interface {
	// List 管理端: 列表 根据交互来, 先是未处理，然后是通过，最后是拒绝
	List(ctx context.Context, feedBack domain.Feedback, offset, limit int) ([]domain.Feedback, error)
	// PendingCount 未处理的数量
	PendingCount(ctx context.Context) (int64, error)
	// Info 详情
	Info(ctx context.Context, id int64) (domain.Feedback, error)
	// UpdateStatus 处理 反馈
	UpdateStatus(ctx context.Context, id int64, status domain.FeedbackStatus) error
	// Create c端: 添加
	Create(ctx context.Context, feedback domain.Feedback) error
}

type feedbackRepo struct {
	dao dao.FeedbackDAO
}

func NewFeedbackRepository(feedBackDao dao.FeedbackDAO) FeedbackRepository {
	return &feedbackRepo{
		dao: feedBackDao,
	}
}

func (f *feedbackRepo) List(ctx context.Context, feedBack domain.Feedback, offset, limit int) ([]domain.Feedback, error) {
	feedBackList, err := f.dao.List(ctx, f.toEntity(feedBack), offset, limit)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.Feedback, 0, len(feedBackList))
	for _, feedBack := range feedBackList {
		ans = append(ans, f.toDomain(feedBack))
	}
	return ans, err
}

func (f *feedbackRepo) PendingCount(ctx context.Context) (int64, error) {
	return f.dao.PendingCount(ctx)
}

func (f *feedbackRepo) Info(ctx context.Context, id int64) (domain.Feedback, error) {
	feedBack, err := f.dao.Info(ctx, id)
	return f.toDomain(feedBack), err
}

func (f *feedbackRepo) UpdateStatus(ctx context.Context, id int64, status domain.FeedbackStatus) error {
	return f.dao.UpdateStatus(ctx, id, int32(status))
}

func (f *feedbackRepo) Create(ctx context.Context, feedback domain.Feedback) error {
	return f.dao.Create(ctx, f.toEntity(feedback))
}

func (f *feedbackRepo) toDomain(feedBack dao.Feedback) domain.Feedback {
	return domain.Feedback{
		ID:      feedBack.ID,
		Biz:     feedBack.Biz,
		BizID:   feedBack.BizID,
		Utime:   time.UnixMilli(feedBack.Utime),
		Ctime:   time.UnixMilli(feedBack.Ctime),
		UID:     feedBack.UID,
		Content: feedBack.Content,
		Status:  domain.FeedbackStatus(feedBack.Status),
	}
}

func (f *feedbackRepo) toEntity(feedBack domain.Feedback) dao.Feedback {
	return dao.Feedback{
		ID:      feedBack.ID,
		Biz:     feedBack.Biz,
		BizID:   feedBack.BizID,
		Utime:   feedBack.Utime.UnixMilli(),
		Ctime:   feedBack.Ctime.UnixMilli(),
		Content: feedBack.Content,
		UID:     feedBack.UID,
		Status:  int32(feedBack.Status),
	}
}
