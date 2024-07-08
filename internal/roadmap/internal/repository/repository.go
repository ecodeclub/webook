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

	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"github.com/ecodeclub/webook/internal/roadmap/internal/repository/dao"
)

var ErrRoadmapNotFound = dao.ErrRecordNotFound

type Repository interface {
	GetByBiz(ctx context.Context, biz string, bizId int64) (domain.Roadmap, error)
}

var _ Repository = &CachedRepository{}

type CachedRepository struct {
	converter
	dao dao.RoadmapDAO
}

func (repo *CachedRepository) GetByBiz(ctx context.Context, biz string, bizId int64) (domain.Roadmap, error) {
	r, err := repo.dao.GetByBiz(ctx, biz, bizId)
	if err != nil {
		return domain.Roadmap{}, err
	}
	edges, err := repo.dao.GetEdgesByRid(ctx, r.Id)
	if err != nil {
		return domain.Roadmap{}, err
	}
	res := repo.toDomain(r)
	res.Edges = repo.edgesToDomain(edges)
	return res, nil
}

func NewCachedRepository(dao dao.RoadmapDAO) Repository {
	return &CachedRepository{dao: dao}
}
