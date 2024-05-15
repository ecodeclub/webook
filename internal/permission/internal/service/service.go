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

	"github.com/ecodeclub/webook/internal/permission/internal/domain"
	"github.com/ecodeclub/webook/internal/permission/internal/repository"
)

type Service interface {
	CreatePersonalPermission(ctx context.Context, ps []domain.PersonalPermission) error
	HasPersonalPermission(ctx context.Context, p domain.PersonalPermission) (bool, error)
	FindPersonalPermissions(ctx context.Context, uid int64) (map[string][]domain.PersonalPermission, error)
}

type permissionService struct {
	repo repository.PermissionRepository
}

func NewPermissionService(repo repository.PermissionRepository) Service {
	return &permissionService{repo: repo}
}

func (s *permissionService) CreatePersonalPermission(ctx context.Context, ps []domain.PersonalPermission) error {
	return s.repo.CreatePersonalPermission(ctx, ps)
}

func (s *permissionService) HasPersonalPermission(ctx context.Context, p domain.PersonalPermission) (bool, error) {
	return s.repo.HasPersonalPermission(ctx, p)
}

func (s *permissionService) FindPersonalPermissions(ctx context.Context, uid int64) (map[string][]domain.PersonalPermission, error) {
	ps, err := s.repo.FindPersonalPermissions(ctx, uid)
	if err != nil {
		return nil, err
	}
	res := make(map[string][]domain.PersonalPermission)
	for _, p := range ps {
		res[p.Biz] = append(res[p.Biz], p)
	}
	return res, err
}
