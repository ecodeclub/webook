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

	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"github.com/ecodeclub/webook/internal/roadmap/internal/repository"
)

var ErrRoadmapNotFound = repository.ErrRoadmapNotFound

type Service interface {
	Detail(ctx context.Context, biz string, bizId int64) (domain.Roadmap, error)
}

var _ Service = &service{}

type service struct {
	repo repository.Repository
}

func (svc *service) Detail(ctx context.Context, biz string, bizId int64) (domain.Roadmap, error) {
	return svc.repo.GetByBiz(ctx, biz, bizId)
}

func NewService(repo repository.Repository) Service {
	return &service{repo: repo}
}
