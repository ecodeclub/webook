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
	"errors"

	"github.com/ecodeclub/webook/internal/interactive/internal/domain"
	"github.com/ecodeclub/webook/internal/interactive/internal/repository"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -source=./interactive.go -destination=../../mocks/interactive.mock.go -package=intrmocks -typed InteractiveService
type Service interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	// LikeToggle 如果点赞过，就取消点赞，如果没点赞过，就点赞
	LikeToggle(c context.Context, biz string, id int64, uid int64) error
	// CollectToggle 如果收藏过，就取消收藏，如果没收藏过，就收藏
	CollectToggle(ctx context.Context, biz string, bizId, uid int64) error
	Get(ctx context.Context, biz string, id int64, uid int64) (domain.Interactive, error)
	GetByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Interactive, error)
}

type interactiveService struct {
	repo repository.InteractiveRepository
}

func NewService(repo repository.InteractiveRepository) Service {
	return &interactiveService{
		repo: repo,
	}
}

func (i *interactiveService) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	return i.repo.IncrViewCnt(ctx, biz, bizId)
}

func (i *interactiveService) LikeToggle(c context.Context, biz string, id int64, uid int64) error {
	return i.repo.LikeToggle(c, biz, id, uid)
}

func (i *interactiveService) CollectToggle(ctx context.Context, biz string, bizId, uid int64) error {
	return i.repo.CollectToggle(ctx, biz, bizId, uid)
}

func (i *interactiveService) Get(ctx context.Context, biz string, id int64, uid int64) (domain.Interactive, error) {
	intr, err := i.repo.Get(ctx, biz, id)
	if err != nil {
		if errors.Is(err, repository.ErrRecordNotFound) {
			return domain.Interactive{}, nil
		}
		return domain.Interactive{}, err
	}
	var eg errgroup.Group
	eg.Go(func() error {
		var er error
		intr.Liked, er = i.repo.Liked(ctx, biz, id, uid)
		return er
	})
	eg.Go(func() error {
		var er error
		intr.Collected, er = i.repo.Collected(ctx, biz, id, uid)
		return er
	})
	return intr, eg.Wait()
}

func (i *interactiveService) GetByIds(ctx context.Context, biz string, ids []int64) (map[int64]domain.Interactive, error) {
	intrs, err := i.repo.GetByIds(ctx, biz, ids)
	if err != nil {
		return nil, err
	}
	intrMap := make(map[int64]domain.Interactive, len(ids))
	for _, intr := range intrs {
		intrMap[intr.BizId] = intr
	}
	for _, id := range ids {
		if _, ok := intrMap[id]; !ok {
			intrMap[id] = domain.Interactive{
				Biz:   biz,
				BizId: id,
			}
		}
	}
	return intrMap, nil

}
