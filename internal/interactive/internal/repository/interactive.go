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
	"errors"

	"github.com/ecodeclub/webook/internal/interactive/internal/domain"
	"github.com/ecodeclub/webook/internal/interactive/internal/repository/dao"
)

var ErrRecordNotFound = dao.ErrRecordNotFound

type InteractiveRepository interface {
	IncrViewCnt(ctx context.Context, biz string, bizId int64) error
	LikeToggle(ctx context.Context, biz string, id int64, uid int64) error
	CollectToggle(ctx context.Context, biz string, id int64, uid int64) error
	Get(ctx context.Context, biz string, id int64) (domain.Interactive, error)
	GetByIds(ctx context.Context, biz string, ids []int64) ([]domain.Interactive, error)
	Liked(ctx context.Context, biz string, id int64, uid int64) (bool, error)
	Collected(ctx context.Context, biz string, id int64, uid int64) (bool, error)
}

type interactiveRepository struct {
	interactiveDao dao.InteractiveDAO
}

func (i *interactiveRepository) IncrViewCnt(ctx context.Context, biz string, bizId int64) error {
	return i.interactiveDao.IncrViewCnt(ctx, biz, bizId)
}

func (i *interactiveRepository) Liked(ctx context.Context, biz string, id int64, uid int64) (bool, error) {
	_, err := i.interactiveDao.GetLikeInfo(ctx, biz, id, uid)
	switch err {
	case nil:
		return true, nil
	case dao.ErrRecordNotFound:
		return false, nil
	default:
		return false, err
	}
}

func (i *interactiveRepository) Collected(ctx context.Context, biz string, id int64, uid int64) (bool, error) {
	_, err := i.interactiveDao.GetCollectInfo(ctx, biz, id, uid)
	switch err {
	case nil:
		return true, nil
	case dao.ErrRecordNotFound:
		return false, nil
	default:
		return false, err
	}
}

func (i *interactiveRepository) LikeToggle(ctx context.Context, biz string, id int64, uid int64) error {
	return i.interactiveDao.LikeToggle(ctx, biz, id, uid)
}

func (i *interactiveRepository) CollectToggle(ctx context.Context, biz string, id int64, uid int64) error {
	return i.interactiveDao.CollectionToggle(ctx, dao.UserCollectionBiz{
		Biz:   biz,
		Uid:   uid,
		BizId: id,
	})
}

func (i *interactiveRepository) Get(ctx context.Context, biz string, id int64) (domain.Interactive, error) {
	intr, err := i.interactiveDao.Get(ctx, biz, id)
	if err != nil {
		if errors.Is(err, dao.ErrRecordNotFound) {
			return domain.Interactive{}, ErrRecordNotFound
		}
		return domain.Interactive{}, err
	}
	return i.toDomain(intr), nil
}

func (i *interactiveRepository) GetByIds(ctx context.Context, biz string, ids []int64) ([]domain.Interactive, error) {
	intrs, err := i.interactiveDao.GetByIds(ctx, biz, ids)
	if err != nil {
		return []domain.Interactive{}, err
	}
	list := make([]domain.Interactive, 0, len(intrs))
	for _, intr := range intrs {
		list = append(list, i.toDomain(intr))
	}
	return list, nil
}

func NewCachedInteractiveRepository(interactiveDao dao.InteractiveDAO) InteractiveRepository {
	return &interactiveRepository{interactiveDao: interactiveDao}
}

func (i *interactiveRepository) toDomain(ie dao.Interactive) domain.Interactive {
	return domain.Interactive{
		Biz:        ie.Biz,
		BizId:      ie.BizId,
		LikeCnt:    ie.LikeCnt,
		CollectCnt: ie.CollectCnt,
		ViewCnt:    ie.ViewCnt,
	}
}
