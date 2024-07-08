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

	"github.com/ecodeclub/ekit/sqlx"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"github.com/ecodeclub/webook/internal/roadmap/internal/repository/dao"
	"golang.org/x/sync/errgroup"
)

type AdminRepository interface {
	Save(ctx context.Context, r domain.Roadmap) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Roadmap, error)
	GetById(ctx context.Context, id int64) (domain.Roadmap, error)
	AddEdge(ctx context.Context, rid int64, edge domain.Edge) error
	DeleteEdge(ctx context.Context, id int64) error
}

var _ AdminRepository = &CachedAdminRepository{}

// CachedAdminRepository 虽然还没缓存，但是将来肯定要有缓存的
type CachedAdminRepository struct {
	converter
	dao dao.AdminDAO
}

func (repo *CachedAdminRepository) DeleteEdge(ctx context.Context, id int64) error {
	return repo.dao.DeleteEdge(ctx, id)
}

func (repo *CachedAdminRepository) AddEdge(ctx context.Context, rid int64, edge domain.Edge) error {
	return repo.dao.AddEdge(ctx, dao.Edge{
		Rid:    rid,
		SrcId:  edge.Src.BizId,
		SrcBiz: edge.Src.Biz.Biz,
		DstId:  edge.Dst.BizId,
		DstBiz: edge.Dst.Biz.Biz,
	})
}

func (repo *CachedAdminRepository) GetById(ctx context.Context, id int64) (domain.Roadmap, error) {
	var (
		eg    errgroup.Group
		r     dao.Roadmap
		edges []dao.Edge
	)
	eg.Go(func() error {
		var err error
		r, err = repo.dao.GetById(ctx, id)
		return err
	})

	eg.Go(func() error {
		var err error
		edges, err = repo.dao.GetEdgesByRid(ctx, id)
		return err
	})
	err := eg.Wait()
	if err != nil {
		return domain.Roadmap{}, err
	}
	res := repo.toDomain(r)
	res.Edges = repo.edgesToDomain(edges)
	return res, nil
}

func (repo *CachedAdminRepository) List(ctx context.Context, offset int, limit int) ([]domain.Roadmap, error) {
	rs, err := repo.dao.List(ctx, offset, limit)
	return slice.Map(rs, func(idx int, src dao.Roadmap) domain.Roadmap {
		return repo.toDomain(src)
	}), err
}

func (repo *CachedAdminRepository) Save(ctx context.Context, r domain.Roadmap) (int64, error) {
	return repo.dao.Save(ctx, repo.toEntity(r))
}

func (repo *CachedAdminRepository) toEntity(r domain.Roadmap) dao.Roadmap {
	return dao.Roadmap{
		Id:    r.Id,
		Title: r.Title,
		Biz:   sqlx.NewNullString(r.Biz),
		BizId: sqlx.NewNullInt64(r.BizId),
	}
}

func NewCachedAdminRepository(dao dao.AdminDAO) AdminRepository {
	return &CachedAdminRepository{
		dao: dao,
	}
}
