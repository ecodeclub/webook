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

	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
)

type Service interface {
	// Save 保存数据，question 绝对不会为 nil
	Save(ctx context.Context, question *domain.Question) (int64, error)
	Publish(ctx context.Context, que *domain.Question) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Question, int64, error)

	PubList(ctx context.Context, offset int, limit int) ([]domain.Question, int64, error)
	Detail(ctx context.Context, qid int64) (domain.Question, error)
	PubDetail(ctx context.Context, qid int64) (domain.Question, error)
}

type service struct {
	repo repository.Repository
}

func (s *service) PubDetail(ctx context.Context, qid int64) (domain.Question, error) {
	return s.repo.GetPubByID(ctx, qid)
}

func (s *service) Detail(ctx context.Context, qid int64) (domain.Question, error) {
	return s.repo.GetById(ctx, qid)
}

func (s *service) List(ctx context.Context, offset int, limit int) ([]domain.Question, int64, error) {
	var (
		eg    errgroup.Group
		qs    []domain.Question
		total int64
	)
	eg.Go(func() error {
		var err error
		qs, err = s.repo.List(ctx, offset, limit)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.Total(ctx)
		return err
	})
	return qs, total, eg.Wait()
}

func (s *service) PubList(ctx context.Context, offset int, limit int) ([]domain.Question, int64, error) {
	var (
		eg    errgroup.Group
		qs    []domain.Question
		total int64
	)
	eg.Go(func() error {
		var err error
		qs, err = s.repo.PubList(ctx, offset, limit)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.PubTotal(ctx)
		return err
	})
	return qs, total, eg.Wait()
}

func (s *service) Save(ctx context.Context, question *domain.Question) (int64, error) {
	if question.Id > 0 {
		return question.Id, s.repo.Update(ctx, question)
	}
	return s.repo.Create(ctx, question)
}

func (s *service) Publish(ctx context.Context, question *domain.Question) (int64, error) {
	return s.repo.Sync(ctx, question)
}

func NewService(repo repository.Repository) Service {
	return &service{repo: repo}
}
