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

	"github.com/ecodeclub/webook/internal/material/internal/domain"
	"github.com/ecodeclub/webook/internal/material/internal/repository"
	"golang.org/x/sync/errgroup"
)

type MaterialService interface {
	Submit(ctx context.Context, m domain.Material) (int64, error)
	FindByID(ctx context.Context, id int64) (domain.Material, error)
	Accept(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]domain.Material, int64, error)
}

type materialService struct {
	repo repository.MaterialRepository
}

func NewMaterialService(repo repository.MaterialRepository) MaterialService {
	return &materialService{
		repo: repo,
	}
}

func (s *materialService) Submit(ctx context.Context, m domain.Material) (int64, error) {
	return s.repo.Create(ctx, m)
}

func (s *materialService) FindByID(ctx context.Context, id int64) (domain.Material, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *materialService) Accept(ctx context.Context, id int64) error {
	return s.repo.Accept(ctx, id)
}

func (s *materialService) List(ctx context.Context, offset int, limit int) ([]domain.Material, int64, error) {
	var (
		materials []domain.Material
		total     int64
		eg        errgroup.Group
	)
	// 并发执行两个查询
	eg.Go(func() error {
		var err error
		materials, err = s.repo.FindAll(ctx, offset, limit)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.CountAll(ctx)
		return err
	})
	// 转换并返回结果
	return materials, total, eg.Wait()
}
