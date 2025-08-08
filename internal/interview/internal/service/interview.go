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
	"errors"

	"github.com/ecodeclub/webook/internal/interview/internal/domain"
	"github.com/ecodeclub/webook/internal/interview/internal/repository"
	"golang.org/x/sync/errgroup"
)

// InterviewService 定义了面试历程相关的业务服务接口。
type InterviewService interface {
	// Save 创建或更新一个新的面试历程。
	Save(ctx context.Context, journey domain.InterviewJourney) (int64, []int64, error)
	// Detail 获取一个完整的面试历程详情，包含所有的面试轮次。
	Detail(ctx context.Context, id, uid int64) (domain.InterviewJourney, error)
	// List 获取一个用户的所有面试历程列表（不包含轮次信息以优化性能）。
	List(ctx context.Context, uid int64, offset, limit int) ([]domain.InterviewJourney, int64, error)
	// FindRoundsByJidAndUid 根据面试历程ID和uid查找所有轮次
	FindRoundsByJidAndUid(ctx context.Context, jid, uid int64) ([]domain.InterviewRound, error)
}

type interviewService struct {
	repo repository.InterviewRepository
}

func NewInterviewService(repo repository.InterviewRepository) InterviewService {
	return &interviewService{repo: repo}
}

func (s *interviewService) Save(ctx context.Context, journey domain.InterviewJourney) (int64, []int64, error) {
	if journey.ID > 0 && len(journey.Rounds) > 0 {
		unshared := make(map[int64]struct{})
		for i := range journey.Rounds {
			if journey.Rounds[i].ID != 0 && !journey.Rounds[i].IsShared() {
				unshared[journey.Rounds[i].ID] = struct{}{}
			}
		}
		if len(unshared) > 0 {
			rounds, err := s.FindRoundsByJidAndUid(ctx, journey.ID, journey.Uid)
			if err != nil {
				return 0, nil, err
			}
			for i := range rounds {
				if _, ok := unshared[rounds[i].ID]; ok && rounds[i].IsShared() {
					return 0, nil, errors.New("不可撤销授权")
				}
			}
		}
	}
	return s.repo.Save(ctx, journey)
}

func (s *interviewService) Detail(ctx context.Context, id, uid int64) (domain.InterviewJourney, error) {
	return s.repo.FindJourneyByID(ctx, id, uid)
}

func (s *interviewService) List(ctx context.Context, uid int64, offset, limit int) ([]domain.InterviewJourney, int64, error) {
	var (
		journeys []domain.InterviewJourney
		total    int64
	)
	var eg errgroup.Group

	eg.Go(func() error {
		var err error
		journeys, err = s.repo.FindJourneysByUID(ctx, uid, offset, limit)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.CountJourneyByUID(ctx, uid)
		return err
	})
	return journeys, total, eg.Wait()
}

func (s *interviewService) FindRoundsByJidAndUid(ctx context.Context, jid, uid int64) ([]domain.InterviewRound, error) {
	return s.repo.FindRoundsByJidAndUid(ctx, jid, uid)
}
