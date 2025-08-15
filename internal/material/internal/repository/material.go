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

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/material/internal/domain"
	"github.com/ecodeclub/webook/internal/material/internal/repository/dao"
)

// MaterialRepository 是素材模块的仓库接口
type MaterialRepository interface {
	Create(ctx context.Context, m domain.Material) (int64, error)
	FindByID(ctx context.Context, id int64) (domain.Material, error)
	UpdateStatus(ctx context.Context, id int64, status domain.MaterialStatus) error
	// Find  分页查询素材列表，如果uid == 0 则在dao层不以uid作为查询条件
	Find(ctx context.Context, uid int64, offset, limit int) ([]domain.Material, error)
	// Count 统计素材列表总数，如果uid == 0 则在dao层不以uid作为查询条件
	Count(ctx context.Context, uid int64) (int64, error)
}

// materialRepository 是 MaterialRepository 的实现
type materialRepository struct {
	dao dao.MaterialDAO
}

func NewMaterialRepository(d dao.MaterialDAO) MaterialRepository {
	return &materialRepository{dao: d}
}

func (r *materialRepository) Create(ctx context.Context, m domain.Material) (int64, error) {
	return r.dao.Create(ctx, r.toEntity(m))
}

func (r *materialRepository) FindByID(ctx context.Context, id int64) (domain.Material, error) {
	material, err := r.dao.FindByID(ctx, id)
	if err != nil {
		return domain.Material{}, err
	}
	return r.toDomain(material), nil
}

func (r *materialRepository) UpdateStatus(ctx context.Context, id int64, status domain.MaterialStatus) error {
	return r.dao.UpdateStatus(ctx, id, status.String())
}

func (r *materialRepository) Find(ctx context.Context, uid int64, offset, limit int) ([]domain.Material, error) {
	materials, err := r.dao.Find(ctx, uid, offset, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map(materials, func(_ int, src dao.Material) domain.Material {
		return r.toDomain(src)
	}), nil
}

func (r *materialRepository) Count(ctx context.Context, uid int64) (int64, error) {
	return r.dao.Count(ctx, uid)
}

func (r *materialRepository) toDomain(m dao.Material) domain.Material {
	return domain.Material{
		ID:        m.ID,
		Uid:       m.Uid,
		Title:     m.Title,
		AudioURL:  m.AudioURL,
		ResumeURL: m.ResumeURL,
		Remark:    m.Remark,
		Status:    domain.MaterialStatus(m.Status),
		Ctime:     m.Ctime,
		Utime:     m.Utime,
	}
}

func (r *materialRepository) toEntity(m domain.Material) dao.Material {
	return dao.Material{
		ID:        m.ID,
		Uid:       m.Uid,
		Title:     m.Title,
		AudioURL:  m.AudioURL,
		ResumeURL: m.ResumeURL,
		Remark:    m.Remark,
		Status:    m.Status.String(),
	}
}
