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
	"database/sql"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/interview/internal/domain"
	"github.com/ecodeclub/webook/internal/interview/internal/repository/dao"
)

// InterviewRepository 定义了面试历程聚合根的仓储接口
type InterviewRepository interface {
	// Save 是一个事务性操作，原子性地保存整个面试历程聚合（包括所有轮次）
	Save(ctx context.Context, journey domain.InterviewJourney) (int64, []int64, error)
	// FindJourneyByID 根据ID查找并完整重建一个面试历程聚合（包含其所有的Rounds）。
	FindJourneyByID(ctx context.Context, id, uid int64) (domain.InterviewJourney, error)
	// FindJourneysByUID 查找一个用户的所有面试历程（不包含Rounds以提高性能）。
	FindJourneysByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.InterviewJourney, error)
	// CountJourneyByUID 计算一个用户的面试历程总数。
	CountJourneyByUID(ctx context.Context, uid int64) (int64, error)
	// FindRoundsByJidAndUid 根据Journey ID和uid查找全部面试轮次
	FindRoundsByJidAndUid(ctx context.Context, jid, uid int64) ([]domain.InterviewRound, error)
}

// interviewRepository 重构后的实现，聚合了两个DAO和DB实例用于事务
type interviewRepository struct {
	dao dao.InterviewDAO
}

func NewInterviewRepository(interviewDAO dao.InterviewDAO) InterviewRepository {
	return &interviewRepository{
		dao: interviewDAO,
	}
}

func (r *interviewRepository) Save(ctx context.Context, journey domain.InterviewJourney) (int64, []int64, error) {
	j, rounds := r.toJourneyEntity(journey)
	return r.dao.Save(ctx, j, rounds)
}

func (r *interviewRepository) toJourneyEntity(j domain.InterviewJourney) (dao.InterviewJourney, []dao.InterviewRound) {
	journey := dao.InterviewJourney{
		ID:          j.ID,
		Uid:         j.Uid,
		CompanyID:   sql.Null[int64]{V: j.CompanyID, Valid: j.CompanyID != 0},
		CompanyName: j.CompanyName,
		JobInfo:     j.JobInfo,
		ResumeURL:   j.ResumeURL,
		Stime:       j.Stime,
		Status:      j.Status.String(),
		Etime:       j.Etime,
	}
	return journey, slice.Map(j.Rounds, func(_ int, src domain.InterviewRound) dao.InterviewRound {
		return r.toRoundEntity(src)
	})
}

func (r *interviewRepository) toRoundEntity(rd domain.InterviewRound) dao.InterviewRound {
	return dao.InterviewRound{
		ID:            rd.ID,
		Uid:           rd.Uid,
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

func (r *interviewRepository) FindJourneyByID(ctx context.Context, id, uid int64) (domain.InterviewJourney, error) {
	journey, rounds, err := r.dao.Find(ctx, id, uid)
	if err != nil {
		return domain.InterviewJourney{}, err
	}
	return r.toJourneyDomain(journey, rounds), nil
}

func (r *interviewRepository) toJourneyDomain(j dao.InterviewJourney, rounds []dao.InterviewRound) domain.InterviewJourney {
	var companyID int64
	if j.CompanyID.Valid {
		companyID = j.CompanyID.V
	}
	return domain.InterviewJourney{
		ID:          j.ID,
		Uid:         j.Uid,
		CompanyID:   companyID,
		CompanyName: j.CompanyName,
		JobInfo:     j.JobInfo,
		ResumeURL:   j.ResumeURL,
		Status:      domain.JourneyStatus(j.Status),
		Stime:       j.Stime,
		Etime:       j.Etime,
		Rounds: slice.Map(rounds, func(_ int, src dao.InterviewRound) domain.InterviewRound {
			return r.toRoundDomain(src)
		}),
	}
}

func (r *interviewRepository) toRoundDomain(rd dao.InterviewRound) domain.InterviewRound {
	return domain.InterviewRound{
		ID:            rd.ID,
		Uid:           rd.Uid,
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

// FindJourneysByUID 查找列表，通常返回不包含完整聚合的“瘦”对象
func (r *interviewRepository) FindJourneysByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.InterviewJourney, error) {
	daoJourneys, err := r.dao.FindJourneysByUID(ctx, uid, offset, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map(daoJourneys, func(_ int, src dao.InterviewJourney) domain.InterviewJourney {
		return r.toJourneyDomain(src, nil)
	}), nil
}

func (r *interviewRepository) CountJourneyByUID(ctx context.Context, uid int64) (int64, error) {
	return r.dao.CountJourneyByUID(ctx, uid)
}

func (r *interviewRepository) FindRoundsByJidAndUid(ctx context.Context, jid, uid int64) ([]domain.InterviewRound, error) {
	found, err := r.dao.FindRoundsByJidAndUid(ctx, jid, uid)
	if err != nil {
		return nil, err
	}
	return slice.Map(found, func(_ int, src dao.InterviewRound) domain.InterviewRound {
		return r.toRoundDomain(src)
	}), nil
}
