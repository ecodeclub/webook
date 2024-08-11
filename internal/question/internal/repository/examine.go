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
	"errors"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
)

type ExamineRepository interface {
	SaveResult(ctx context.Context, uid, qid int64, result domain.ExamineResult) error
	GetResultByUidAndQid(ctx context.Context, uid int64, qid int64) (domain.Result, error)
	GetResultsByIds(ctx context.Context, uid int64, ids []int64) ([]domain.ExamineResult, error)
	UpdateQuestionResult(ctx context.Context, uid int64, qid int64, result domain.Result) error
}

var _ ExamineRepository = &CachedExamineRepository{}

type CachedExamineRepository struct {
	dao dao.ExamineDAO
}

func (repo *CachedExamineRepository) GetResultsByIds(ctx context.Context, uid int64, ids []int64) ([]domain.ExamineResult, error) {
	res, err := repo.dao.GetResultByUidAndQids(ctx, uid, ids)
	return slice.Map(res, func(idx int, src dao.QuestionResult) domain.ExamineResult {
		return domain.ExamineResult{
			Qid:    src.Qid,
			Result: domain.Result(src.Result),
		}
	}), err
}

func (repo *CachedExamineRepository) GetResultByUidAndQid(ctx context.Context, uid int64, qid int64) (domain.Result, error) {
	res, err := repo.dao.GetResultByUidAndQid(ctx, uid, qid)
	if errors.Is(err, dao.ErrRecordNotFound) {
		return domain.ResultFailed, nil
	}
	return domain.Result(res.Result), err
}

func (repo *CachedExamineRepository) SaveResult(ctx context.Context, uid, qid int64, result domain.ExamineResult) error {
	// 开始记录
	err := repo.dao.SaveResult(ctx, dao.ExamineRecord{
		Uid:       uid,
		Qid:       qid,
		Tid:       result.Tid,
		Result:    result.Result.ToUint8(),
		RawResult: result.RawResult,
		Tokens:    result.Tokens,
		Amount:    result.Amount,
	})
	return err
}

func (repo *CachedExamineRepository) UpdateQuestionResult(ctx context.Context, uid int64, qid int64, result domain.Result) error {
	err := repo.dao.UpdateQuestionResult(ctx, dao.QuestionResult{
		Uid:    uid,
		Qid:    qid,
		Result: result.ToUint8(),
	})
	return err
}

func NewCachedExamineRepository(dao dao.ExamineDAO) ExamineRepository {
	return &CachedExamineRepository{dao: dao}
}
