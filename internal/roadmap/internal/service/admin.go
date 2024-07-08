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

type AdminService interface {
	Detail(ctx context.Context, id int64) (domain.Roadmap, error)
	Save(ctx context.Context, r domain.Roadmap) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Roadmap, error)
	AddEdge(ctx context.Context, rid int64, edge domain.Edge) error
	DeleteEdge(ctx context.Context, id int64) error
}

var _ AdminService = &adminService{}

type adminService struct {
	repo repository.AdminRepository
}

func (svc *adminService) DeleteEdge(ctx context.Context, id int64) error {
	return svc.repo.DeleteEdge(ctx, id)
}

func (svc *adminService) AddEdge(ctx context.Context, rid int64, edge domain.Edge) error {
	return svc.repo.AddEdge(ctx, rid, edge)
}

func (svc *adminService) Detail(ctx context.Context, id int64) (domain.Roadmap, error) {
	return svc.repo.GetById(ctx, id)
}

func (svc *adminService) List(ctx context.Context, offset int, limit int) ([]domain.Roadmap, error) {
	return svc.repo.List(ctx, offset, limit)
}

func (svc *adminService) Save(ctx context.Context, r domain.Roadmap) (int64, error) {
	return svc.repo.Save(ctx, r)
}

func NewAdminService(repo repository.AdminRepository) AdminService {
	return &adminService{
		repo: repo,
	}
}
