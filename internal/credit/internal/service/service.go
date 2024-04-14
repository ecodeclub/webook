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
	"fmt"

	"github.com/ecodeclub/webook/internal/credit/internal/domain"
	"github.com/ecodeclub/webook/internal/credit/internal/repository"
)

var (
	ErrCreditNotEnough = errors.New("积分不足")
)

//go:generate mockgen -source=./service.go -destination=../../mocks/credit.mock.go -package=creditmocks Service
type Service interface {
	AddCredits(ctx context.Context, credit domain.Credit) error
	GetCreditsByUID(ctx context.Context, uid int64) (domain.Credit, error)
	TryDeductCredits(ctx context.Context, credit domain.Credit) (id int64, err error)
	ConfirmDeductCredits(ctx context.Context, uid, tid int64) error
	CancelDeductCredits(ctx context.Context, uid, tid int64) error
}

func NewService(repo repository.CreditRepository) Service {
	return &service{repo: repo}
}

type service struct {
	repo repository.CreditRepository
}

func NewCreditService(repo repository.CreditRepository) Service {
	return &service{repo: repo}
}

func (s *service) AddCredits(ctx context.Context, credit domain.Credit) error {
	return s.repo.AddCredits(ctx, credit)
}

func (s *service) GetCreditsByUID(ctx context.Context, uid int64) (domain.Credit, error) {
	return s.repo.GetCreditByUID(ctx, uid)
}

func (s *service) TryDeductCredits(ctx context.Context, credit domain.Credit) (id int64, err error) {
	c, err := s.repo.GetCreditByUID(ctx, credit.Uid)
	if err != nil {
		return 0, err
	}
	if credit.ChangeAmount > c.TotalAmount {
		return 0, fmt.Errorf("%w", ErrCreditNotEnough)
	}
	return s.repo.TryDeductCredits(ctx, credit)
}

func (s *service) ConfirmDeductCredits(ctx context.Context, uid, tid int64) error {
	return s.repo.ConfirmDeductCredits(ctx, uid, tid)
}

func (s *service) CancelDeductCredits(ctx context.Context, uid, tid int64) error {
	return s.repo.CancelDeductCredits(ctx, uid, tid)
}
