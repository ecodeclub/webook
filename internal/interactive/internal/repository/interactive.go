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
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/interactive/internal/domain"
	"github.com/ecodeclub/webook/internal/interactive/internal/repository/dao"
)

const (
	CaseBiz        = "case"
	CaseSetBiz     = "caseSet"
	QuestionBiz    = "question"
	QuestionSetBiz = "questionSet"
)

var defaultTimeout = 1 * time.Second
var ErrRecordNotFound = dao.ErrRecordNotFound

type InteractiveRepository interface {
	IncrViewCnt(ctx context.Context, biz string, bizId int64) error
	LikeToggle(ctx context.Context, biz string, id int64, uid int64) error
	CollectToggle(ctx context.Context, biz string, id int64, uid int64) error
	Get(ctx context.Context, biz string, id int64) (domain.Interactive, error)
	GetByIds(ctx context.Context, biz string, uid int64, ids []int64) ([]domain.Interactive, error)
	Liked(ctx context.Context, biz string, id int64, uid int64) (bool, error)
	Collected(ctx context.Context, biz string, id int64, uid int64) (bool, error)

	// 保存收藏夹
	SaveCollection(ctx context.Context, collection domain.Collection) (int64, error)
	// 删除收藏夹
	DeleteCollection(ctx context.Context, uid, collectionId int64) error
	// 收藏夹列表
	CollectionList(ctx context.Context, uid int64, offset, limit int) ([]domain.Collection, error)
	// CollectionInfo 收藏夹收藏记录
	CollectionInfo(ctx context.Context, uid, collectionId int64, offset, limit int) ([]domain.CollectionRecord, error)
	// MoveCollection 转移收藏夹
	MoveCollection(ctx context.Context, biz string, bizid, uid, collectionId int64) error
}

type interactiveRepository struct {
	interactiveDao dao.InteractiveDAO
	logger         *elog.Component
}

func (i *interactiveRepository) MoveCollection(ctx context.Context, biz string, bizid, uid, collectionId int64) error {
	return i.interactiveDao.MoveCollection(ctx, biz, bizid, uid, collectionId)
}

func (i *interactiveRepository) SaveCollection(ctx context.Context, collection domain.Collection) (int64, error) {
	collectionEntity := i.collectionToEntity(collection)
	return i.interactiveDao.SaveCollection(ctx, collectionEntity)
}

func (i *interactiveRepository) DeleteCollection(ctx context.Context, uid, collectionId int64) error {
	// 查询删除收藏夹的详情
	collectionRecords, err := i.interactiveDao.CollectionInfo(ctx, collectionId)
	if err != nil {
		return err
	}
	err = i.interactiveDao.DeleteCollection(ctx, uid, collectionId)
	if err != nil {
		return err
	}
	// 减少计数
	go i.decrCollectionCount(collectionRecords)
	return nil
}

func (i *interactiveRepository) CollectionList(ctx context.Context, uid int64, offset, limit int) ([]domain.Collection, error) {
	clist, err := i.interactiveDao.CollectionList(ctx, uid, offset, limit)
	if err != nil {
		return nil, err
	}
	collections := make([]domain.Collection, 0)
	for _, c := range clist {
		collections = append(collections, i.collectionToDomain(c))
	}
	return collections, nil
}

func (i *interactiveRepository) CollectionInfo(ctx context.Context, uid, collectionId int64, offset, limit int) ([]domain.CollectionRecord, error) {
	userBizs, err := i.interactiveDao.CollectionInfoWithPage(ctx, uid, collectionId, offset, limit)
	if err != nil {
		return nil, err
	}
	records := make([]domain.CollectionRecord, 0, len(userBizs))
	for _, userBiz := range userBizs {
		record := i.toCollectionRecord(userBiz)
		records = append(records, record)
	}
	return records, nil
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
	return i.interactiveDao.CollectToggle(ctx, dao.UserCollectionBiz{
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

func (i *interactiveRepository) GetByIds(ctx context.Context, biz string, uid int64, ids []int64) ([]domain.Interactive, error) {
	var (
		intrs        []dao.Interactive
		likedMap     = map[int64]struct{}{}
		collectedMap = map[int64]struct{}{}
		eg           errgroup.Group
	)
	eg.Go(func() error {
		var eerr error
		intrs, eerr = i.interactiveDao.GetByIds(ctx, biz, ids)
		return eerr
	})

	eg.Go(func() error {
		var eerr error
		likeds, eerr := i.interactiveDao.GetUserLikes(ctx, uid, biz, ids)
		if eerr != nil {
			return eerr
		}
		for _, liked := range likeds {
			likedMap[liked.BizId] = struct{}{}
		}
		return eerr
	})

	eg.Go(func() error {
		var eerr error
		collecteds, eerr := i.interactiveDao.GetUserCollects(ctx, uid, biz, ids)
		if eerr != nil {
			return eerr
		}
		for _, collected := range collecteds {
			collectedMap[collected.BizId] = struct{}{}
		}
		return eerr
	})

	err := eg.Wait()
	if err != nil {
		return nil, err
	}
	list := make([]domain.Interactive, 0, len(intrs))
	for _, intr := range intrs {
		domainIntr := i.toDomain(intr)
		_, collected := collectedMap[domainIntr.BizId]
		domainIntr.Collected = collected
		_, liked := likedMap[domainIntr.BizId]
		domainIntr.Liked = liked
		list = append(list, domainIntr)
	}
	return list, nil
}

func NewCachedInteractiveRepository(interactiveDao dao.InteractiveDAO) InteractiveRepository {
	return &interactiveRepository{
		interactiveDao: interactiveDao,
		logger:         elog.DefaultLogger,
	}
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

func (i *interactiveRepository) collectionToDomain(collectionDao dao.Collection) domain.Collection {
	return domain.Collection{
		Id:   collectionDao.Id,
		Uid:  collectionDao.Uid,
		Name: collectionDao.Name,
	}
}

func (i *interactiveRepository) collectionToEntity(ie domain.Collection) dao.Collection {
	return dao.Collection{
		Id:   ie.Id,
		Uid:  ie.Uid,
		Name: ie.Name,
	}
}

func (i *interactiveRepository) toCollectionRecord(collectBiz dao.UserCollectionBiz) domain.CollectionRecord {
	record := domain.CollectionRecord{
		Id:  collectBiz.Id,
		Biz: collectBiz.Biz,
	}
	switch collectBiz.Biz {
	case CaseBiz:
		record.Case = collectBiz.BizId
	case CaseSetBiz:
		record.CaseSet = collectBiz.BizId
	case QuestionBiz:
		record.Question = collectBiz.BizId
	case QuestionSetBiz:
		record.QuestionSet = collectBiz.BizId
	}
	return record

}

func (i *interactiveRepository) decrCollectionCount(collectionRecords []dao.UserCollectionBiz) {
	for _, record := range collectionRecords {
		ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
		err := i.interactiveDao.DecrCollectCount(ctx, record.Biz, record.BizId)
		if err != nil {
			i.logger.Error("减少收藏计数失败", elog.FieldErr(err))
		}
		cancel()
	}
}
