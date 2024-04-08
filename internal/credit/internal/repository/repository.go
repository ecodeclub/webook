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
	c, l := r.toEntity(credit)
	_, err := r.dao.Create(ctx, c, l)
	return err
}

func (r *creditRepository) toEntity(credit domain.Credit) (dao.Credit, dao.CreditLog) {
	c := dao.Credit{
		Id:                 0,
		Uid:                credit.Uid,
		TotalCredits:       0,
		LockedTotalCredits: 0,
		Version:            0,
	}
	l := dao.CreditLog{}
	return c, l
}

func (r *creditRepository) GetCreditByUID(ctx context.Context, uid int64) (domain.Credit, error) {
	byUID, err := r.dao.FindCreditByUID(ctx, uid)
	return r.toDomain(byUID), err
}

func (r *creditRepository) toDomain(d dao.Credit) domain.Credit {
	return domain.Credit{
		Uid:    d.Uid,
		Amount: d.TotalCredits,
	}
}
