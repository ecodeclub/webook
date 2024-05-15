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
	"github.com/ecodeclub/webook/internal/permission/internal/domain"
	"github.com/ecodeclub/webook/internal/permission/internal/repository/dao"
)

type PermissionRepository interface {
	CreatePersonalPermission(ctx context.Context, ps []domain.PersonalPermission) error
	HasPersonalPermission(ctx context.Context, p domain.PersonalPermission) (bool, error)
	FindPersonalPermissions(ctx context.Context, uid int64) ([]domain.PersonalPermission, error)
}

type permissionRepository struct {
	dao dao.PermissionDAO
}

func NewPermissionRepository(dao dao.PermissionDAO) PermissionRepository {
	return &permissionRepository{dao: dao}
}

func (r *permissionRepository) CreatePersonalPermission(ctx context.Context, ps []domain.PersonalPermission) error {
	entities := slice.Map(ps, func(idx int, src domain.PersonalPermission) dao.PersonalPermission {
		return r.toEntity(src)
	})
	return r.dao.CreatePersonalPermission(ctx, entities)
}

func (r *permissionRepository) HasPersonalPermission(ctx context.Context, perm domain.PersonalPermission) (bool, error) {
	count, err := r.dao.CountPersonalPermission(ctx, r.toEntity(perm))
	return count > 0, err
}

func (r *permissionRepository) toEntity(p domain.PersonalPermission) dao.PersonalPermission {
	return dao.PersonalPermission{
		UID:   p.UID,
		Biz:   p.Biz,
		BizId: p.BizID,
	}
}

func (r *permissionRepository) FindPersonalPermissions(ctx context.Context, uid int64) ([]domain.PersonalPermission, error) {
	ps, err := r.dao.FindPersonalPermissions(ctx, uid)
	if err != nil {
		return nil, err
	}
	return slice.Map(ps, func(idx int, src dao.PersonalPermission) domain.PersonalPermission {
		return r.toDomain(src)
	}), err
}

func (r *permissionRepository) toDomain(p dao.PersonalPermission) domain.PersonalPermission {
	return domain.PersonalPermission{
		UID:   p.UID,
		Biz:   p.Biz,
		BizID: p.BizId,
	}
}
