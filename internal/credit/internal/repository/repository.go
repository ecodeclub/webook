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

type CreditRepository interface {
	AddCredits(ctx context.Context, credit domain.Credit) error
	GetCreditByUID(ctx context.Context, uid int64) (domain.Credit, error)
}

type creditRepository struct {
	dao dao.CreditDAO
}

func NewCreditRepository(dao dao.CreditDAO) CreditRepository {
	return &creditRepository{dao: dao}
}

func (r *creditRepository) AddCredits(ctx context.Context, credit domain.Credit) error {
	cl := r.toCreditLogsEntity(credit.Logs)
	_, err := r.dao.Upsert(ctx, credit.Uid, credit.ChangeAmount, cl[0])
	return err
}

func (r *creditRepository) toCreditLogsEntity(cl []domain.CreditLog) []dao.CreditLog {
	return slice.Map(cl, func(idx int, src domain.CreditLog) dao.CreditLog {
		return dao.CreditLog{
			BizId:   src.BizId,
			BizType: src.BizType,
			Desc:    src.Action,
			Status:  src.Status,
		}
	})
}

func (r *creditRepository) GetCreditByUID(ctx context.Context, uid int64) (domain.Credit, error) {
	c, err := r.dao.FindCreditByUID(ctx, uid)
	if err != nil {
		return domain.Credit{}, err
	}
	cl, err := r.dao.FindCreditLogsByCreditID(ctx, c.Id)
	return r.toDomain(c, cl), err
}

func (r *creditRepository) toDomain(d dao.Credit, l []dao.CreditLog) domain.Credit {
	return domain.Credit{
		Uid:         d.Uid,
		TotalAmount: d.TotalCredits,
		Logs: slice.Map(l, func(idx int, src dao.CreditLog) domain.CreditLog {
			return domain.CreditLog{
				BizId:   src.BizId,
				BizType: src.BizType,
				Action:  src.Desc,
				Status:  src.Status,
			}
		}),
	}
}
