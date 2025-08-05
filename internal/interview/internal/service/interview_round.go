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
)

// InterviewRoundService 定义了面试轮次相关的业务服务接口。
type InterviewRoundService interface {
	// Create 为一个已存在的面试历程添加一个新的面试轮次。
	Create(ctx context.Context, rd domain.InterviewRound) (int64, error)
	// Update 更新一个面试轮次的信息。
	Update(ctx context.Context, rd domain.InterviewRound) error
	// FindByID 根据ID查找面试轮次，用于测试
	FindByID(ctx context.Context, id, jid, uid int64) (domain.InterviewRound, error)
}

type roundService struct {
	journeyRepo repository.InterviewJourneyRepository // 需要 journeyRepo 来做业务校验
	roundRepo   repository.InterviewRoundRepository
}

func NewInterviewRoundService(
	journeyRepo repository.InterviewJourneyRepository,
	roundRepo repository.InterviewRoundRepository,
) InterviewRoundService {
	return &roundService{
		journeyRepo: journeyRepo,
		roundRepo:   roundRepo,
	}
}

func (s *roundService) Create(ctx context.Context, rd domain.InterviewRound) (int64, error) {
	_, err := s.journeyRepo.FindByID(ctx, rd.Jid, rd.Uid)
	if err != nil {
		return 0, err
	}
	return s.roundRepo.Create(ctx, rd)
}

func (s *roundService) FindByID(ctx context.Context, id, jid, uid int64) (domain.InterviewRound, error) {
	return s.roundRepo.FindByID(ctx, id, jid, uid)
}

func (s *roundService) Update(ctx context.Context, rd domain.InterviewRound) error {
	r, err := s.FindByID(ctx, rd.ID, rd.Jid, rd.Uid)
	if err != nil {
		return err
	}
	if r.IsShared() && !rd.IsShared() {
		return errors.New("不可撤销授权")
	}
	return s.roundRepo.Update(ctx, rd)
}
