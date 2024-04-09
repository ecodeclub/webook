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

package service

import (
	"context"
	"fmt"

	"github.com/ecodeclub/webook/internal/feedback/internal/event"
	"github.com/gotomicro/ego/core/elog"
	"github.com/lithammer/shortuuid/v4"

	"github.com/ecodeclub/webook/internal/feedback/internal/domain"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository"
)

type Service interface {
	// List 管理端: 列表 根据交互来
	List(ctx context.Context, feedback domain.Feedback, offset, limit int) ([]domain.Feedback, error)
	// PendingCount 未处理的个数
	PendingCount(ctx context.Context) (int64, error)
	// Info 详情
	Info(ctx context.Context, id int64) (domain.Feedback, error)
	// UpdateStatus 处理 反馈
	UpdateStatus(ctx context.Context, feedback domain.Feedback) error
	// Create c端: 添加
	Create(ctx context.Context, feedback domain.Feedback) error
}

type service struct {
	repo     repository.FeedbackRepository
	producer *event.IncreaseCreditsEventProducer
	logger   *elog.Component
}

func NewFeedbackService(repo repository.FeedbackRepository, producer *event.IncreaseCreditsEventProducer) Service {
	return &service{
		repo:     repo,
		logger:   elog.DefaultLogger,
		producer: producer,
	}
}

func (s *service) PendingCount(ctx context.Context) (int64, error) {
	return s.repo.PendingCount(ctx)
}

func (s *service) Info(ctx context.Context, id int64) (domain.Feedback, error) {
	return s.repo.Info(ctx, id)
}

func (s *service) UpdateStatus(ctx context.Context, feedback domain.Feedback) error {
	info, err := s.repo.Info(ctx, feedback.ID)
	if err != nil {
		return fmt.Errorf("反馈ID非法: %w", err)
	}
	err = s.repo.UpdateStatus(ctx, feedback.ID, feedback.Status)
	if err != nil {
		return err
	}
	if feedback.Status == domain.Adopt {
		evt := event.CreditIncreaseEvent{
			Key:    shortuuid.New(),
			Uid:    info.UID,
			Amount: 100,
			Biz:    9,
			BizId:  info.ID,
			Action: "采纳反馈",
		}
		if er := s.producer.Produce(ctx, evt); er != nil {
			s.logger.Error("发送增加积分消息失败",
				elog.FieldErr(er),
				elog.Any("event", evt),
			)
		}
	}
	return nil
}

func (s *service) Create(ctx context.Context, feedback domain.Feedback) error {
	return s.repo.Create(ctx, feedback)
}

func (s *service) List(ctx context.Context, feedBack domain.Feedback, offset, limit int) ([]domain.Feedback, error) {
	return s.repo.List(ctx, feedBack, offset, limit)
}
