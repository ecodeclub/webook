package dao

import (
	"context"
	"time"

	"github.com/ego-component/egorm"
)

type FeedbackDAO interface {
	// List 列表 根据交互来
	List(ctx context.Context, offset, limit int) ([]Feeback, error)
	// PendingCount 未处理的个数
	PendingCount(ctx context.Context) (int64, error)
	// Info 详情
	Info(ctx context.Context, id int64) (Feeback, error)
	// UpdateStatus 处理 反馈，反馈人的id
	UpdateStatus(ctx context.Context, id int64, status int32) error
	// Create 添加
	Create(ctx context.Context, feedback Feeback) error
}
type feedBackDAO struct {
	db *egorm.Component
}

func NewFeedbackDAO(db *egorm.Component) FeedbackDAO {
	return &feedBackDAO{
		db: db,
	}
}

func (f *feedBackDAO) List(ctx context.Context, offset, limit int) ([]Feeback, error) {
	var res []Feeback
	err := f.db.WithContext(ctx).
		Select("id", "biz_id", "biz", "uid", "status", "utime").
		Order("status asc,id desc").
		Offset(offset).Limit(limit).Find(&res).Error
	return res, err
}

func (f *feedBackDAO) PendingCount(ctx context.Context) (int64, error) {
	var count int64
	builder := f.db.WithContext(ctx).Model(&Feeback{}).Where("status = ?", 0)
	err := builder.Count(&count).Error
	return count, err
}

func (f *feedBackDAO) Info(ctx context.Context, id int64) (Feeback, error) {
	var feedBack Feeback
	err := f.db.WithContext(ctx).Where("id = ?", id).First(&feedBack).Error
	return feedBack, err
}

func (f *feedBackDAO) UpdateStatus(ctx context.Context, id int64, status int32) error {
	err := f.db.WithContext(ctx).
		Model(&Feeback{}).
		Where("id = ?", id).Updates(map[string]any{
		"status": status,
		"utime":  time.Now().UnixMilli(),
	}).Error
	return err

}

func (f *feedBackDAO) Create(ctx context.Context, feedback Feeback) error {
	feedback.Ctime = time.Now().UnixMilli()
	feedback.Utime = time.Now().UnixMilli()
	return f.db.WithContext(ctx).Create(&feedback).Error
}
