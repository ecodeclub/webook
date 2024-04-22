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
	"github.com/ecodeclub/webook/internal/credit/internal/domain"
	"github.com/ecodeclub/webook/internal/credit/internal/repository/dao"
)

var (
	ErrDuplicatedCreditLog = dao.ErrDuplicatedCreditLog
	ErrCreditNotEnough     = dao.ErrCreditNotEnough
	ErrRecordNotFound      = dao.ErrRecordNotFound
)

type CreditRepository interface {
	AddCredits(ctx context.Context, credit domain.Credit) error
	GetCreditByUID(ctx context.Context, uid int64) (domain.Credit, error)
	TryDeductCredits(ctx context.Context, credit domain.Credit) (int64, error)
	ConfirmDeductCredits(ctx context.Context, uid, tid int64) error
	CancelDeductCredits(ctx context.Context, uid, tid int64) error
}

type creditRepository struct {
	dao dao.CreditDAO
}

func NewCreditRepository(dao dao.CreditDAO) CreditRepository {
	return &creditRepository{dao: dao}
}

func (r *creditRepository) AddCredits(ctx context.Context, credit domain.Credit) error {
	cl := r.toCreditLogsEntity(credit)
	err := r.dao.Upsert(ctx, cl[0])
	return err
}

func (r *creditRepository) toCreditLogsEntity(c domain.Credit) []dao.CreditLog {
	return slice.Map(c.Logs, func(idx int, src domain.CreditLog) dao.CreditLog {
		return dao.CreditLog{
			Key:          src.Key,
			Uid:          c.Uid,
			BizId:        src.BizId,
			Biz:          src.Biz,
			Desc:         src.Desc,
			CreditChange: src.ChangeAmount,
		}
	})
}

func (r *creditRepository) GetCreditByUID(ctx context.Context, uid int64) (domain.Credit, error) {
	c, err := r.dao.FindCreditByUID(ctx, uid)
	if err != nil {
		return domain.Credit{}, err
	}
	cl, err := r.dao.FindCreditLogsByUID(ctx, uid)
	return r.toDomain(c, cl), err
}

func (r *creditRepository) toDomain(d dao.Credit, l []dao.CreditLog) domain.Credit {
	return domain.Credit{
		Uid:               d.Uid,
		TotalAmount:       d.TotalCredits,
		LockedTotalAmount: d.LockedTotalCredits,
		Logs: slice.Map(l, func(idx int, src dao.CreditLog) domain.CreditLog {
			return domain.CreditLog{
				ID:           src.Id,
				Uid:          src.Uid,
				Key:          src.Key,
				ChangeAmount: src.CreditChange,
				BizId:        src.BizId,
				Biz:          src.Biz,
				Desc:         src.Desc,
			}
		}),
	}
}

func (r *creditRepository) TryDeductCredits(ctx context.Context, credit domain.Credit) (int64, error) {
	cl := r.toCreditLogsEntity(credit)
	id, err := r.dao.CreateCreditLockLog(ctx, cl[0])
	return id, err
}

func (r *creditRepository) ConfirmDeductCredits(ctx context.Context, uid, tid int64) error {
	return r.dao.ConfirmCreditLockLog(ctx, uid, tid)
}

func (r *creditRepository) CancelDeductCredits(ctx context.Context, uid, tid int64) error {
	return r.dao.CancelCreditLockLog(ctx, uid, tid)
}
