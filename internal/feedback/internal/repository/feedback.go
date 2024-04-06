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
	List(ctx context.Context, feedBack domain.FeedBack, offset, limit int) ([]domain.FeedBack, error)
	// 未处理的数量
	PendingCount(ctx context.Context) (int64, error)
	// 详情
	Info(ctx context.Context, id int64) (domain.FeedBack, error)
	// 处理 反馈
	UpdateStatus(ctx context.Context, id int64, status domain.FeedBackStatus) error
	//	c端
	// 添加
	Create(ctx context.Context, feedback domain.FeedBack) error
}

type feedBackRepo struct {
	feedBackDao dao.FeedBackDAO
}

func NewFeedBackRepo(feedBackDao dao.FeedBackDAO) FeedBackRepo {
	return &feedBackRepo{
		feedBackDao: feedBackDao,
	}
}

func (f *feedBackRepo) List(ctx context.Context, feedBack domain.FeedBack, offset, limit int) ([]domain.FeedBack, error) {
	feedBackList, err := f.feedBackDao.List(ctx, f.toEntity(feedBack), offset, limit)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.FeedBack, 0, len(feedBackList))
	for _, feedBack := range feedBackList {
		ans = append(ans, f.toDomain(feedBack))
	}
	return ans, err
}

func (f *feedBackRepo) PendingCount(ctx context.Context) (int64, error) {
	return f.feedBackDao.PendingCount(ctx)
}

func (f *feedBackRepo) Info(ctx context.Context, id int64) (domain.FeedBack, error) {
	feedBack, err := f.feedBackDao.Info(ctx, id)
	return f.toDomain(feedBack), err
}

func (f *feedBackRepo) UpdateStatus(ctx context.Context, id int64, status domain.FeedBackStatus) error {
	return f.feedBackDao.UpdateStatus(ctx, id, int32(status))
}

func (f *feedBackRepo) Create(ctx context.Context, feedback domain.FeedBack) error {
	return f.feedBackDao.Create(ctx, f.toEntity(feedback))
}

func (f *feedBackRepo) toDomain(feedBack dao.FeedBack) domain.FeedBack {
	return domain.FeedBack{
		ID:      feedBack.ID,
		Biz:     feedBack.Biz,
		BizID:   feedBack.BizID,
		Utime:   time.UnixMilli(feedBack.Utime),
		Ctime:   time.UnixMilli(feedBack.Ctime),
		UID:     feedBack.UID,
		Content: feedBack.Content,
		Status:  domain.FeedBackStatus(feedBack.Status),
	}
}

func (f *feedBackRepo) toEntity(feedBack domain.FeedBack) dao.FeedBack {
	return dao.FeedBack{
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
