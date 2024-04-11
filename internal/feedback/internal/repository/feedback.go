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

package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/feedback/internal/domain"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository/dao"
)

type FeedbackRepository interface {
	// List 管理端: 列表 根据交互来, 先是未处理，然后是通过，最后是拒绝
	List(ctx context.Context, offset, limit int) ([]domain.Feedback, error)
	// PendingCount 未处理的数量
	PendingCount(ctx context.Context) (int64, error)
	// Info 详情
	Info(ctx context.Context, id int64) (domain.Feedback, error)
	// UpdateStatus 处理 反馈
	UpdateStatus(ctx context.Context, id int64, status domain.FeedbackStatus) error
	// Create C端: 添加
	Create(ctx context.Context, feedback domain.Feedback) error
}

type feedbackRepository struct {
	dao dao.FeedbackDAO
}

func NewFeedBackRepository(feedBackDao dao.FeedbackDAO) FeedbackRepository {
	return &feedbackRepository{
		dao: feedBackDao,
	}
}

func (f *feedbackRepository) List(ctx context.Context, offset, limit int) ([]domain.Feedback, error) {
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

func (f *feedbackRepository) PendingCount(ctx context.Context) (int64, error) {
	return f.dao.PendingCount(ctx)
}

func (f *feedbackRepository) Info(ctx context.Context, id int64) (domain.Feedback, error) {
	fb, err := f.dao.Info(ctx, id)
	return f.toDomain(fb), err
}

func (f *feedbackRepository) UpdateStatus(ctx context.Context, id int64, status domain.FeedbackStatus) error {
	return f.dao.UpdateStatus(ctx, id, int32(status))
}

func (f *feedbackRepository) Create(ctx context.Context, feedback domain.Feedback) error {
	return f.dao.Create(ctx, f.toEntity(feedback))
}

func (f *feedbackRepository) toDomain(fb dao.Feedback) domain.Feedback {
	return domain.Feedback{
		ID:      fb.ID,
		Biz:     fb.Biz,
		BizID:   fb.BizID,
		Utime:   time.UnixMilli(fb.Utime),
		Ctime:   time.UnixMilli(fb.Ctime),
		UID:     fb.UID,
		Content: fb.Content,
		Status:  domain.FeedbackStatus(fb.Status),
	}
}

func (f *feedbackRepository) toEntity(feedBack domain.Feedback) dao.Feedback {
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
