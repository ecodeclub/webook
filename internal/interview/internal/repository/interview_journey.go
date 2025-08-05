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

// InterviewJourneyRepository 定义了面试历程聚合根的仓储接口
type InterviewJourneyRepository interface {
	// Create 创建一个新的面试历程聚合。
	Create(ctx context.Context, journey domain.InterviewJourney) (int64, error)
	// Update 保存一个已存在的面试历程聚合的变更。
	Update(ctx context.Context, journey domain.InterviewJourney) error
	// FindByID 根据ID查找并完整重建一个面试历程聚合（包含其所有的Rounds）。
	FindByID(ctx context.Context, id, uid int64) (domain.InterviewJourney, error)
	// FindByUID 查找一个用户的所有面试历程（不包含Rounds以提高性能）。
	FindByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.InterviewJourney, error)
	// CountByUID 计算一个用户的面试历程总数。
	CountByUID(ctx context.Context, uid int64) (int64, error)
}

// journeyRepository 是 InterviewJourneyRepository 的实现
type journeyRepository struct {
	journeyDAO dao.InterviewJourneyDAO
	roundRepo  InterviewRoundRepository
}

// NewInterviewJourneyRepository 创建一个新的面试历程仓储实例
func NewInterviewJourneyRepository(journeyDAO dao.InterviewJourneyDAO, roundRepo InterviewRoundRepository) InterviewJourneyRepository {
	return &journeyRepository{
		journeyDAO: journeyDAO,
		roundRepo:  roundRepo,
	}
}

func (r *journeyRepository) Create(ctx context.Context, journey domain.InterviewJourney) (int64, error) {
	return r.journeyDAO.Create(ctx, r.toEntity(journey))
}

func (r *journeyRepository) Update(ctx context.Context, journey domain.InterviewJourney) error {
	daoJourney := r.toEntity(journey)
	return r.journeyDAO.Update(ctx, daoJourney)
}

func (r *journeyRepository) FindByID(ctx context.Context, id, uid int64) (domain.InterviewJourney, error) {
	daoJourney, err := r.journeyDAO.First(ctx, id, uid)
	if err != nil {
		return domain.InterviewJourney{}, err
	}
	rounds, err := r.roundRepo.FindByJidAndUid(ctx, id, uid)
	if err != nil {
		return domain.InterviewJourney{}, err
	}
	return r.toDomain(daoJourney, rounds), nil
}

// FindByUID 查找列表，通常返回不包含完整聚合的“瘦”对象
func (r *journeyRepository) FindByUID(ctx context.Context, uid int64, offset, limit int) ([]domain.InterviewJourney, error) {
	daoJourneys, err := r.journeyDAO.FindByUID(ctx, uid, offset, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map(daoJourneys, func(_ int, src dao.InterviewJourney) domain.InterviewJourney {
		return r.toDomain(src, nil)
	}), nil
}

func (r *journeyRepository) CountByUID(ctx context.Context, uid int64) (int64, error) {
	return r.journeyDAO.CountByUID(ctx, uid)
}

func (r *journeyRepository) toEntity(j domain.InterviewJourney) dao.InterviewJourney {
	return dao.InterviewJourney{
		Uid:         j.Uid,
		CompanyID:   sql.Null[int64]{V: j.CompanyID, Valid: j.CompanyID != 0},
		CompanyName: j.CompanyName,
		JobInfo:     j.JobInfo,
		ResumeURL:   j.ResumeURL,
		Stime:       j.Stime,
		Status:      j.Status.String(),
		Etime:       j.Etime,
	}
}

func (r *journeyRepository) toDomain(j dao.InterviewJourney, rounds []domain.InterviewRound) domain.InterviewJourney {
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
		Rounds:      rounds,
	}
}
