package dao

import (
	"context"
	"time"

	"github.com/ego-component/egorm"
)

type FeedBackDAO interface {
	// 管理端
	// 列表 根据交互来
	List(ctx context.Context, f FeedBack, offset, limit int) ([]FeedBack, error)
	// 未处理的个数
	PendingCount(ctx context.Context) (int64, error)
	// 详情
	Info(ctx context.Context, id int64) (FeedBack, error)
	// 处理 反馈
	UpdateStatus(ctx context.Context, id int64, status int32) error
	//	c端
	// 添加
	Create(ctx context.Context, feedback FeedBack) error
}
type feedBackDAO struct {
	db *egorm.Component
}

func NewFeedBackDAO(db *egorm.Component) FeedBackDAO {
	return &feedBackDAO{
		db: db,
	}
}

func (f *feedBackDAO) List(ctx context.Context, feedBack FeedBack, offset, limit int) ([]FeedBack, error) {
	var feedBackList []FeedBack
	builder := f.db.WithContext(ctx).
		Select([]string{"id", "biz_id", "biz", "uid", "status", "utime"})
	if feedBack.Biz != "" {
		builder = builder.Where("biz = ?", feedBack.Biz)
		if feedBack.BizID != 0 {
			builder = builder.Where("biz_id = ?", feedBack.BizID)
		}
	}
	err := builder.Order("status asc,id desc").
		Offset(offset).Limit(limit).Find(&feedBackList).Error
	return feedBackList, err
}

func (f *feedBackDAO) PendingCount(ctx context.Context) (int64, error) {
	var count int64
	builder := f.db.WithContext(ctx).Model(&FeedBack{}).Where("status = ?", 0)
	err := builder.Count(&count).Error
	return count, err
}

func (f *feedBackDAO) Info(ctx context.Context, id int64) (FeedBack, error) {
	var feedBack FeedBack
	err := f.db.WithContext(ctx).Where("id = ?", id).First(&feedBack).Error
	return feedBack, err
}

func (f *feedBackDAO) UpdateStatus(ctx context.Context, id int64, status int32) error {
	return f.db.WithContext(ctx).
		Model(&FeedBack{}).
		Where("id = ?", id).Updates(map[string]any{
		"status": status,
		"utime":  time.Now().UnixMilli(),
	}).Error
}

func (f *feedBackDAO) Create(ctx context.Context, feedback FeedBack) error {
	feedback.Ctime = time.Now().UnixMilli()
	feedback.Utime = time.Now().UnixMilli()
	return f.db.WithContext(ctx).Create(&feedback).Error
}
