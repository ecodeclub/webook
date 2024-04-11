// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dao

import (
	"context"
	"time"

	"github.com/ego-component/egorm"
)

type FeedbackDAO interface {
	// List 列表 根据交互来
	List(ctx context.Context, offset, limit int) ([]Feedback, error)
	// PendingCount 未处理的个数
	PendingCount(ctx context.Context) (int64, error)
	// Info 详情
	Info(ctx context.Context, id int64) (Feedback, error)
	// UpdateStatus 处理 反馈，反馈人的id
	UpdateStatus(ctx context.Context, id int64, status int32) error
	// Create 添加
	Create(ctx context.Context, feedback Feedback) error
}
type feedBackDAO struct {
	db *egorm.Component
}

func NewFeedbackDAO(db *egorm.Component) FeedbackDAO {
	return &feedBackDAO{
		db: db,
	}
}

func (f *feedBackDAO) List(ctx context.Context, offset, limit int) ([]Feedback, error) {
	var res []Feedback
	err := f.db.WithContext(ctx).
		Select("id", "biz_id", "biz", "uid", "status", "utime").
		Order("status asc,id desc").
		Offset(offset).Limit(limit).Find(&res).Error
	return res, err
}

func (f *feedBackDAO) PendingCount(ctx context.Context) (int64, error) {
	var count int64
	builder := f.db.WithContext(ctx).Model(&Feedback{}).Where("status = ?", 0)
	err := builder.Count(&count).Error
	return count, err
}

func (f *feedBackDAO) Info(ctx context.Context, id int64) (Feedback, error) {
	var feedBack Feedback
	err := f.db.WithContext(ctx).Where("id = ?", id).First(&feedBack).Error
	return feedBack, err
}

func (f *feedBackDAO) UpdateStatus(ctx context.Context, id int64, status int32) error {
	err := f.db.WithContext(ctx).
		Model(&Feedback{}).
		Where("id = ?", id).Updates(map[string]any{
		"status": status,
		"utime":  time.Now().UnixMilli(),
	}).Error
	return err

}

func (f *feedBackDAO) Create(ctx context.Context, feedback Feedback) error {
	feedback.Ctime = time.Now().UnixMilli()
	feedback.Utime = time.Now().UnixMilli()
	return f.db.WithContext(ctx).Create(&feedback).Error
}

type Feedback struct {
	ID      int64  `gorm:"primaryKey,autoIncrement"`
	BizID   int64  `gorm:"column:biz_id;type:int;comment:业务ID;not null;index:idx_biz_biz_id;default:0"`
	Biz     string `gorm:"column:biz;type:varchar(255);comment:业务名称;not null;index:idx_biz_biz_id;default:''"`
	UID     int64  `gorm:"column:uid;type:bigint;comment:用户ID;not null;default:0"`
	Content string `gorm:"column:content;type:text;comment:内容;"`
	Status  int32  `gorm:"column:status;type:tinyint(3);default:0;index:idx_status;comment:状态 0-未处理 1-采纳 2-拒绝;not null"`
	Ctime   int64
	Utime   int64
}
