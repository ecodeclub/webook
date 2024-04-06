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

	"github.com/ecodeclub/webook/internal/label/internal/domain"
	"github.com/ecodeclub/webook/internal/label/internal/repository"
)

const systemUid int64 = -1

type Service interface {
	SystemLabels(ctx context.Context) ([]domain.Label, error)
	CreateSystemLabel(ctx context.Context, name string) (int64, error)
}

type service struct {
	repo repository.LabelRepository
}

func (s *service) CreateSystemLabel(ctx context.Context, name string) (int64, error) {
	return s.repo.CreateLabel(ctx, systemUid, name)
}

func (s *service) SystemLabels(ctx context.Context) ([]domain.Label, error) {
	return s.repo.UidLabels(ctx, systemUid)
}

func NewService(repo repository.LabelRepository) Service {
	return &service{repo: repo}
}
