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

	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/feedback/internal/domain"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository"
)

type Service interface {
	// List 管理端: 列表 根据交互来
	List(ctx context.Context, feedBack domain.FeedBack, offset, limit int) ([]domain.FeedBack, error)
	// PendingCount 未处理的个数
	PendingCount(ctx context.Context) (int64, error)
	// Info 详情
	Info(ctx context.Context, id int64) (domain.FeedBack, error)
	// UpdateStatus 处理 反馈
	UpdateStatus(ctx context.Context, domainFeedback domain.FeedBack) error
	// Create c端: 添加
	Create(ctx context.Context, feedback domain.FeedBack) error
}

type service struct {
	repo repository.FeedbackRepository
	// creditsEventProducer *event.CreditsEventProducer
	logger *elog.Component
}

func NewService(repo repository.FeedbackRepository) Service {
	return &service{
		repo:   repo,
		logger: elog.DefaultLogger,
	}
}

func (s *service) PendingCount(ctx context.Context) (int64, error) {
	return s.repo.PendingCount(ctx)
}

func (s *service) Info(ctx context.Context, id int64) (domain.FeedBack, error) {
	return s.repo.Info(ctx, id)
}

func (s *service) UpdateStatus(ctx context.Context, domainFeedback domain.FeedBack) error {
	err := s.repo.UpdateStatus(ctx, domainFeedback.ID, domainFeedback.Status)
	if err != nil {
		return err
	}
	// todo 添加发送反馈成功时间
	// if domainFeedback.Status == domain.Access {

	// evt := event.CreditsEvent{
	//	Uid: uid,
	// }
	// if eerr := s.creditsEventProducer.Produce(ctx, evt); eerr != nil {
	//	s.logger.Error("发送反馈成功消息失败",
	//		elog.FieldErr(eerr),
	//		elog.FieldKey("event"),
	//		elog.FieldValueAny(evt),
	//	)
	// }
	// }
	return nil
}

func (s *service) Create(ctx context.Context, feedback domain.FeedBack) error {
	return s.repo.Create(ctx, feedback)
}

func (s *service) List(ctx context.Context, feedBack domain.FeedBack, offset, limit int) ([]domain.FeedBack, error) {
	return s.repo.List(ctx, feedBack, offset, limit)
}
