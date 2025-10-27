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

	"github.com/ecodeclub/ekit/slice"
	baguwen "github.com/ecodeclub/webook/internal/question"

	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"github.com/ecodeclub/webook/internal/roadmap/internal/repository"
)

type AdminService interface {
	Detail(ctx context.Context, id int64) (domain.Roadmap, error)
	Save(ctx context.Context, r domain.Roadmap) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Roadmap, error)
	Delete(ctx context.Context, id int64) error

	SanitizeData()

	SaveNode(ctx context.Context, node domain.Node) (int64, error)
	DeleteNode(ctx context.Context, id int64) error
	NodeList(ctx context.Context, rid int64) ([]domain.Node, error)
	SaveEdge(ctx context.Context, rid int64, edge domain.Edge) error
	DeleteEdge(ctx context.Context, id int64) error
}

var _ AdminService = &adminService{}

type adminService struct {
	repo      repository.AdminRepository
	queSetSvc baguwen.QuestionSetService
}

func (svc *adminService) Delete(ctx context.Context, id int64) error {
	return svc.repo.Delete(ctx, id)
}

func (svc *adminService) SanitizeData() {
	svc.repo.SanitizeData()
}

func (svc *adminService) SaveNode(ctx context.Context, node domain.Node) (int64, error) {
	return svc.repo.SaveNode(ctx, node)
}

func (svc *adminService) DeleteNode(ctx context.Context, id int64) error {
	return svc.repo.DeleteNode(ctx, id)
}

func (svc *adminService) NodeList(ctx context.Context, rid int64) ([]domain.Node, error) {
	return svc.repo.NodeList(ctx, rid)
}

func (svc *adminService) SaveEdge(ctx context.Context, rid int64, edge domain.Edge) error {
	return svc.repo.SaveEdgeV1(ctx, rid, edge)
}

func (svc *adminService) DeleteEdge(ctx context.Context, id int64) error {
	return svc.repo.DeleteEdgeV1(ctx, id)
}

func (svc *adminService) Detail(ctx context.Context, id int64) (domain.Roadmap, error) {
	return svc.repo.GetById(ctx, id)
}

func (svc *adminService) List(ctx context.Context, offset int, limit int) ([]domain.Roadmap, error) {
	return svc.repo.List(ctx, offset, limit)
}

func (svc *adminService) Save(ctx context.Context, r domain.Roadmap) (int64, error) {
	id, err := svc.repo.Save(ctx, r)
	if err != nil {
		return 0, err
	}
	// 新增并且路线图是题集的，新建题目节点
	if r.Biz == domain.BizQuestionSet && r.Id == 0 {
		qs, err := svc.queSetSvc.Detail(ctx, r.BizId)
		if err != nil {
			return 0, err
		}
		nodes := slice.Map(qs.Questions, func(idx int, src baguwen.Question) domain.Node {
			return domain.Node{
				Biz: domain.Biz{
					Biz:   domain.BizQuestion,
					BizId: src.Id,
				},
				Rid: id,
			}
		})
		err = svc.repo.SaveNodes(ctx, nodes)
	}
	return id, err
}

func NewAdminService(repo repository.AdminRepository, queSetSvc baguwen.QuestionSetService) AdminService {
	return &adminService{
		repo:      repo,
		queSetSvc: queSetSvc,
	}
}
