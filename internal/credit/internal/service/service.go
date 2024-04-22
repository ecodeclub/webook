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
	"golang.org/x/sync/errgroup"
)

var (
	ErrCreditNotEnough     = repository.ErrCreditNotEnough
	ErrDuplicatedCreditLog = repository.ErrDuplicatedCreditLog
	ErrInvalidCreditLog    = errors.New("积分流水信息非法")
	ErrRecordNotFound      = repository.ErrRecordNotFound
)

//go:generate mockgen -source=./service.go -destination=../../mocks/credit.mock.go -package=creditmocks -typed Service
type Service interface {
	AddCredits(ctx context.Context, credit domain.Credit) error
	GetCreditsByUID(ctx context.Context, uid int64) (domain.Credit, error)
	TryDeductCredits(ctx context.Context, credit domain.Credit) (id int64, err error)
	ConfirmDeductCredits(ctx context.Context, uid, tid int64) error
	CancelDeductCredits(ctx context.Context, uid, tid int64) error
	FindExpiredLockedCreditLogs(ctx context.Context, offset int, limit int, ctime int64) ([]domain.CreditLog, int64, error)
}

type service struct {
	repo repository.CreditRepository
}

func NewCreditService(repo repository.CreditRepository) Service {
	return &service{repo: repo}
}

func (s *service) AddCredits(ctx context.Context, credit domain.Credit) error {
	if len(credit.Logs) != 1 {
		return fmt.Errorf("%w", ErrInvalidCreditLog)
	}
	return s.repo.AddCredits(ctx, credit)
}

func (s *service) GetCreditsByUID(ctx context.Context, uid int64) (domain.Credit, error) {
	c, err := s.repo.GetCreditByUID(ctx, uid)
	if errors.Is(err, ErrRecordNotFound) {
		return domain.Credit{Uid: uid}, nil
	}
	return c, err
}

func (s *service) TryDeductCredits(ctx context.Context, credit domain.Credit) (id int64, err error) {
	if len(credit.Logs) != 1 {
		return 0, fmt.Errorf("%w", ErrInvalidCreditLog)
	}
	_, err = s.repo.GetCreditByUID(ctx, credit.Uid)
	if err != nil {
		return 0, err
	}
	return s.repo.TryDeductCredits(ctx, credit)
}

func (s *service) ConfirmDeductCredits(ctx context.Context, uid, tid int64) error {
	return s.repo.ConfirmDeductCredits(ctx, uid, tid)
}

func (s *service) CancelDeductCredits(ctx context.Context, uid, tid int64) error {
	return s.repo.CancelDeductCredits(ctx, uid, tid)
}

func (s *service) FindExpiredLockedCreditLogs(ctx context.Context, offset int, limit int, ctime int64) ([]domain.CreditLog, int64, error) {
	var (
		eg    errgroup.Group
		cs    []domain.CreditLog
		total int64
	)
	eg.Go(func() error {
		var err error
		cs, err = s.repo.FindExpiredLockedCreditLogs(ctx, offset, limit, ctime)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.TotalExpiredLockedCreditLogs(ctx, ctime)
		return err
	})
	return cs, total, eg.Wait()
}
