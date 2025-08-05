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

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/interview/internal/domain"
	"github.com/ecodeclub/webook/internal/interview/internal/repository/dao"
)

// InterviewRoundRepository 定义了面试轮次实体的仓储接口。
// 它的职责是解耦 Domain 层与 DAO 层，处理两者之间的数据转换。
type InterviewRoundRepository interface {
	// Create 创建一个新的面试轮次。
	Create(ctx context.Context, round domain.InterviewRound) (int64, error)
	// Update 更新一个已有的面试轮次。
	Update(ctx context.Context, round domain.InterviewRound) error
	// FindByID 根据ID, JID和UID精确查找一个面试轮次。
	FindByID(ctx context.Context, id, jid, uid int64) (domain.InterviewRound, error)
	// FindByJidAndUid 查找一个面试历程下的所有轮次。
	FindByJidAndUid(ctx context.Context, jid, uid int64) ([]domain.InterviewRound, error)
}

type roundRepository struct {
	dao dao.InterviewRoundDAO
}

// NewInterviewRoundRepository 创建一个新的面试轮次仓储实例
func NewInterviewRoundRepository(d dao.InterviewRoundDAO) InterviewRoundRepository {
	return &roundRepository{dao: d}
}

func (r *roundRepository) Create(ctx context.Context, round domain.InterviewRound) (int64, error) {
	return r.dao.Create(ctx, r.toEntity(round))
}

func (r *roundRepository) Update(ctx context.Context, round domain.InterviewRound) error {
	return r.dao.Update(ctx, r.toEntity(round))
}

func (r *roundRepository) FindByID(ctx context.Context, id, jid, uid int64) (domain.InterviewRound, error) {
	found, err := r.dao.First(ctx, id, jid, uid)
	if err != nil {
		return domain.InterviewRound{}, err
	}
	return r.toDomain(found), nil
}

func (r *roundRepository) FindByJidAndUid(ctx context.Context, jid, uid int64) ([]domain.InterviewRound, error) {
	found, err := r.dao.FindByJidAndUid(ctx, jid, uid)
	if err != nil {
		return nil, err
	}
	return slice.Map(found, func(_ int, src dao.InterviewRound) domain.InterviewRound {
		return r.toDomain(src)
	}), nil
}

func (r *roundRepository) toEntity(rd domain.InterviewRound) dao.InterviewRound {
	return dao.InterviewRound{
		ID:            rd.ID,
		Uid:           rd.Uid,
		Jid:           rd.Jid,
		RoundNumber:   rd.RoundNumber,
		RoundType:     rd.RoundType,
		InterviewDate: rd.InterviewDate,
		JobInfo:       rd.JobInfo,
		ResumeURL:     rd.ResumeURL,
		AudioURL:      rd.AudioURL,
		SelfResult:    rd.SelfResult,
		SelfSummary:   rd.SelfSummary,
		Result:        rd.Result.String(),
		AllowSharing:  rd.AllowSharing,
	}
}

func (r *roundRepository) toDomain(rd dao.InterviewRound) domain.InterviewRound {
	return domain.InterviewRound{
		ID:            rd.ID,
		Uid:           rd.Uid,
		Jid:           rd.Jid,
		RoundNumber:   rd.RoundNumber,
		RoundType:     rd.RoundType,
		InterviewDate: rd.InterviewDate,
		JobInfo:       rd.JobInfo,
		ResumeURL:     rd.ResumeURL,
		AudioURL:      rd.AudioURL,
		SelfResult:    rd.SelfResult,
		SelfSummary:   rd.SelfSummary,
		Result:        domain.RoundResult(rd.Result),
		AllowSharing:  rd.AllowSharing,
	}
}
