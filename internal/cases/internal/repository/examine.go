package repository

import (
	"context"
	"errors"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"

	"github.com/ecodeclub/ekit/slice"
)

type ExamineRepository interface {
	SaveResult(ctx context.Context, uid, cid int64, result domain.ExamineCaseResult) error
	GetResultByUidAndQid(ctx context.Context, uid int64, cid int64) (domain.CaseResult, error)
	GetResultsByIds(ctx context.Context, uid int64, ids []int64) ([]domain.ExamineCaseResult, error)
}

var _ ExamineRepository = &CachedExamineRepository{}

type CachedExamineRepository struct {
	dao dao.ExamineDAO
}

func (repo *CachedExamineRepository) GetResultsByIds(ctx context.Context, uid int64, ids []int64) ([]domain.ExamineCaseResult, error) {
	res, err := repo.dao.GetResultByUidAndCids(ctx, uid, ids)
	return slice.Map(res, func(idx int, src dao.CaseResult) domain.ExamineCaseResult {
		return domain.ExamineCaseResult{
			Cid:    src.Cid,
			Result: domain.CaseResult(src.Result),
		}
	}), err
}

func (repo *CachedExamineRepository) GetResultByUidAndQid(ctx context.Context, uid int64, cid int64) (domain.CaseResult, error) {
	res, err := repo.dao.GetResultByUidAndCid(ctx, uid, cid)
	if errors.Is(err, dao.ErrRecordNotFound) {
		return domain.ResultFailed, nil
	}
	return domain.CaseResult(res.Result), err
}

func (repo *CachedExamineRepository) SaveResult(ctx context.Context, uid, cid int64, result domain.ExamineCaseResult) error {
	// 开始记录
	err := repo.dao.SaveResult(ctx, dao.CaseExamineRecord{
		Uid:       uid,
		Cid:       cid,
		Tid:       result.Tid,
		Result:    result.Result.ToUint8(),
		RawResult: result.RawResult,
		Tokens:    result.Tokens,
		Amount:    result.Amount,
	})
	return err
}

func NewCachedExamineRepository(dao dao.ExamineDAO) ExamineRepository {
	return &CachedExamineRepository{dao: dao}
}
