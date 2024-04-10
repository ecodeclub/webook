package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/feedback/internal/domain"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository/dao"
)

type FeedBackRepo interface {
	// 管理端
	// 列表 根据交互来, 先是未处理，然后是通过，最后是拒绝
	List(ctx context.Context, offset, limit int) ([]domain.Feedback, error)
	// 未处理的数量
	PendingCount(ctx context.Context) (int64, error)
	// 详情
	Info(ctx context.Context, id int64) (domain.Feedback, error)
	// 处理 反馈
	UpdateStatus(ctx context.Context, id int64, status domain.FeedBackStatus) error
	//	c端
	// 添加
	Create(ctx context.Context, feedback domain.Feedback) error
}

type feedBackRepo struct {
	dao dao.FeedbackDAO
}

func NewFeedBackRepo(feedBackDao dao.FeedbackDAO) FeedBackRepo {
	return &feedBackRepo{
		dao: feedBackDao,
	}
}

func (f *feedBackRepo) List(ctx context.Context, offset, limit int) ([]domain.Feedback, error) {
	feedBackList, err := f.dao.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.Feedback, 0, len(feedBackList))
	for _, feedBack := range feedBackList {
		ans = append(ans, f.toDomain(feedBack))
	}
	return ans, err
}

func (f *feedBackRepo) PendingCount(ctx context.Context) (int64, error) {
	return f.dao.PendingCount(ctx)
}

func (f *feedBackRepo) Info(ctx context.Context, id int64) (domain.Feedback, error) {
	fb, err := f.dao.Info(ctx, id)
	return f.toDomain(fb), err
}

func (f *feedBackRepo) UpdateStatus(ctx context.Context, id int64, status domain.FeedBackStatus) error {
	return f.dao.UpdateStatus(ctx, id, int32(status))
}

func (f *feedBackRepo) Create(ctx context.Context, feedback domain.Feedback) error {
	return f.dao.Create(ctx, f.toEntity(feedback))
}

func (f *feedBackRepo) toDomain(fb dao.Feeback) domain.Feedback {
	return domain.Feedback{
		ID:      fb.ID,
		Biz:     fb.Biz,
		BizID:   fb.BizID,
		Utime:   time.UnixMilli(fb.Utime),
		Ctime:   time.UnixMilli(fb.Ctime),
		UID:     fb.UID,
		Content: fb.Content,
		Status:  domain.FeedBackStatus(fb.Status),
	}
}

func (f *feedBackRepo) toEntity(feedBack domain.Feedback) dao.Feeback {
	return dao.Feeback{
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
