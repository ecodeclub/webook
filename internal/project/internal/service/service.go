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

	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ecodeclub/webook/internal/project/internal/repository"
)

// Service C 端接口
type Service interface {
	List(ctx context.Context, offset int, limit int) ([]domain.Project, error)
	Detail(ctx context.Context, id int64) (domain.Project, error)
}

var _ Service = &service{}

type service struct {
	repo repository.Repository
}

func (s *service) Detail(ctx context.Context, id int64) (domain.Project, error) {
	return s.repo.Detail(ctx, id)
}

func (s *service) List(ctx context.Context, offset int, limit int) ([]domain.Project, error) {
	return s.repo.List(ctx, offset, limit)
}

func NewService(repo repository.Repository) Service {
	return &service{repo: repo}
}
