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

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"golang.org/x/sync/errgroup"
)

type QuestionSetService interface {
	Create(ctx context.Context, set domain.QuestionSet) (int64, error)
	UpdateQuestions(ctx context.Context, set domain.QuestionSet) error
	List(ctx context.Context, offset, limit int, uid int64) ([]domain.QuestionSet, int64, error)
	Detail(ctx context.Context, id, uid int64) (domain.QuestionSet, error)
}

type questionSetService struct {
	repo repository.QuestionSetRepository
}

func NewQuestionSetService(repo repository.QuestionSetRepository) QuestionSetService {
	return &questionSetService{repo: repo}
}

func (q *questionSetService) Create(ctx context.Context, set domain.QuestionSet) (int64, error) {
	return q.repo.Create(ctx, set)
}

func (q *questionSetService) UpdateQuestions(ctx context.Context, set domain.QuestionSet) error {
	return q.repo.UpdateQuestions(ctx, set)
}

func (q *questionSetService) Detail(ctx context.Context, id, uid int64) (domain.QuestionSet, error) {
	return q.repo.GetByIDAndUID(ctx, id, uid)
}

func (q *questionSetService) List(ctx context.Context, offset, limit int, uid int64) ([]domain.QuestionSet, int64, error) {
	var (
		eg    errgroup.Group
		qs    []domain.QuestionSet
		total int64
	)
	eg.Go(func() error {
		var err error
		qs, err = q.repo.List(ctx, offset, limit, uid)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = q.repo.Total(ctx, uid)
		return err
	})
	return qs, total, eg.Wait()
}
