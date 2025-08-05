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

	"github.com/ecodeclub/webook/internal/interview/internal/domain"
	"github.com/ecodeclub/webook/internal/interview/internal/repository"
	"golang.org/x/sync/errgroup"
)

// InterviewJourneyService 定义了面试历程相关的业务服务接口。
type InterviewJourneyService interface {
	// Create 创建一个新的面试历程。
	Create(ctx context.Context, j domain.InterviewJourney) (int64, error)
	// Update 更新一个面试历程的信息。
	Update(ctx context.Context, j domain.InterviewJourney) error
	// Detail 获取一个完整的面试历程详情，包含所有的面试轮次。
	Detail(ctx context.Context, id, uid int64) (domain.InterviewJourney, error)
	// List 获取一个用户的所有面试历程列表（不包含轮次信息以优化性能）。
	List(ctx context.Context, uid int64, offset, limit int) ([]domain.InterviewJourney, int64, error)
}

type journeyService struct {
	repo repository.InterviewJourneyRepository
}

func NewInterviewJourneyService(repo repository.InterviewJourneyRepository) InterviewJourneyService {
	return &journeyService{repo: repo}
}

func (s *journeyService) Create(ctx context.Context, j domain.InterviewJourney) (int64, error) {
	return s.repo.Create(ctx, j)
}

func (s *journeyService) Update(ctx context.Context, j domain.InterviewJourney) error {
	_, err := s.repo.FindByID(ctx, j.ID, j.Uid)
	if err != nil {
		return err
	}
	return s.repo.Update(ctx, j)
}

func (s *journeyService) Detail(ctx context.Context, id, uid int64) (domain.InterviewJourney, error) {
	return s.repo.FindByID(ctx, id, uid)
}

func (s *journeyService) List(ctx context.Context, uid int64, offset, limit int) ([]domain.InterviewJourney, int64, error) {
	var (
		journeys []domain.InterviewJourney
		total    int64
	)
	var eg errgroup.Group

	eg.Go(func() error {
		var err error
		journeys, err = s.repo.FindByUID(ctx, uid, offset, limit)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.CountByUID(ctx, uid)
		return err
	})
	return journeys, total, eg.Wait()
}
