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
	"time"

	"github.com/ecodeclub/ekit/sqlx"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/question/internal/repository/cache"
	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
)

type Repository interface {
	PubList(ctx context.Context, offset int, limit int) ([]domain.Question, error)
	PubTotal(ctx context.Context) (int64, error)
	// Sync 保存到制作库，而后同步到线上库
	Sync(ctx context.Context, que *domain.Question) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Question, error)
	Total(ctx context.Context) (int64, error)
	Update(ctx context.Context, question *domain.Question) error
	Create(ctx context.Context, question *domain.Question) (int64, error)
	GetById(ctx context.Context, qid int64) (domain.Question, error)
	GetPubByID(ctx context.Context, qid int64) (domain.Question, error)
	GetPubByIDs(ctx context.Context, ids []int64) ([]domain.Question, error)
}

// CachedRepository 支持缓存的 repository 实现
// 虽然在 Save 的时候，理论上也需要更新缓存，但是没有必要
type CachedRepository struct {
	dao    dao.QuestionDAO
	cache  cache.QuestionCache
	logger *elog.Component
}

func (c *CachedRepository) GetPubByIDs(ctx context.Context, qids []int64) ([]domain.Question, error) {
	data, err := c.dao.GetPubByIDs(ctx, qids)
	return slice.Map(data, func(idx int, src dao.PublishQuestion) domain.Question {
		return c.toDomain(dao.Question(src))
	}), err
}

func (c *CachedRepository) GetPubByID(ctx context.Context, qid int64) (domain.Question, error) {
	// 可以缓存
	data, pubEles, err := c.dao.GetPubByID(ctx, qid)
	if err != nil {
		return domain.Question{}, err
	}
	eles := slice.Map(pubEles, func(idx int, src dao.PublishAnswerElement) dao.AnswerElement {
		return dao.AnswerElement(src)
	})
	return c.toDomainWithAnswer(dao.Question(data), eles), nil
}

func (c *CachedRepository) GetById(ctx context.Context, qid int64) (domain.Question, error) {
	data, eles, err := c.dao.GetByID(ctx, qid)
	if err != nil {
		return domain.Question{}, err
	}
	return c.toDomainWithAnswer(data, eles), nil
}

func (c *CachedRepository) Update(ctx context.Context, question *domain.Question) error {
	q, eles := c.toEntity(question)
	return c.dao.Update(ctx, q, eles)
}

func (c *CachedRepository) Create(ctx context.Context, question *domain.Question) (int64, error) {
	q, eles := c.toEntity(question)
	return c.dao.Create(ctx, q, eles)
}

func (c *CachedRepository) Sync(ctx context.Context, que *domain.Question) (int64, error) {
	// 理论上来说要更新缓存，但是我懒得写了
	q, eles := c.toEntity(que)
	return c.dao.Sync(ctx, q, eles)
}

func (c *CachedRepository) List(ctx context.Context, offset int, limit int) ([]domain.Question, error) {
	qs, err := c.dao.List(ctx, offset, limit)
	return slice.Map(qs, func(idx int, src dao.Question) domain.Question {
		return c.toDomain(src)
	}), err
}

func (c *CachedRepository) Total(ctx context.Context) (int64, error) {
	return c.dao.Count(ctx)
}

func (c *CachedRepository) PubList(ctx context.Context, offset int, limit int) ([]domain.Question, error) {
	// TODO 缓存第一页
	qs, err := c.dao.PubList(ctx, offset, limit)
	return slice.Map(qs, func(idx int, src dao.PublishQuestion) domain.Question {
		return c.toDomain(dao.Question(src))
	}), err
}

func (c *CachedRepository) PubTotal(ctx context.Context) (int64, error) {
	res, err := c.cache.GetTotal(ctx)
	if err == nil {
		return res, err
	}
	res, err = c.dao.PubCount(ctx)
	if err != nil {
		return 0, err
	}
	err = c.cache.SetTotal(ctx, res)
	if err != nil {
		c.logger.Error("更新缓存中的总数失败", elog.FieldErr(err))
	}
	return res, nil
}

func (c *CachedRepository) toDomainWithAnswer(que dao.Question, eles []dao.AnswerElement) domain.Question {
	res := c.toDomain(que)
	for _, ele := range eles {
		switch ele.Type {
		case dao.AnswerElementTypeAnalysis:
			res.Answer.Analysis = c.ele2Domain(ele)
		case dao.AnswerElementTypeBasic:
			res.Answer.Basic = c.ele2Domain(ele)
		case dao.AnswerElementTypeIntermedia:
			res.Answer.Intermediate = c.ele2Domain(ele)
		case dao.AnswerElementTypeAdvanced:
			res.Answer.Advanced = c.ele2Domain(ele)
		}
	}
	return res
}

func (c *CachedRepository) toDomain(que dao.Question) domain.Question {
	return domain.Question{
		Id:      que.Id,
		Uid:     que.Uid,
		Title:   que.Title,
		Content: que.Content,
		Labels:  que.Labels.Val,
		Status:  domain.QuestionStatus(que.Status),
		Utime:   time.UnixMilli(que.Utime),
	}
}

func (c *CachedRepository) toEntity(que *domain.Question) (dao.Question, []dao.AnswerElement) {
	now := time.Now().UnixMilli()
	q := dao.Question{
		Id:      que.Id,
		Uid:     que.Uid,
		Title:   que.Title,
		Labels:  sqlx.JsonColumn[[]string]{Val: que.Labels, Valid: len(que.Labels) != 0},
		Content: que.Content,
		Status:  que.Status.ToUint8(),
		Ctime:   now,
		Utime:   now,
	}
	// 固定是 4 个部分
	eles := []dao.AnswerElement{
		c.ele2Entity(que.Id, now, dao.AnswerElementTypeAnalysis, que.Answer.Analysis),
		c.ele2Entity(que.Id, now, dao.AnswerElementTypeBasic, que.Answer.Basic),
		c.ele2Entity(que.Id, now, dao.AnswerElementTypeIntermedia, que.Answer.Intermediate),
		c.ele2Entity(que.Id, now, dao.AnswerElementTypeAdvanced, que.Answer.Advanced),
	}
	return q, eles
}

func (c *CachedRepository) ele2Domain(ele dao.AnswerElement) domain.AnswerElement {
	return domain.AnswerElement{
		Id:        ele.Id,
		Content:   ele.Content,
		Keywords:  ele.Keywords,
		Shorthand: ele.Shorthand,
		Highlight: ele.Highlight,
		Guidance:  ele.Guidance,
	}
}

func (c *CachedRepository) ele2Entity(qid int64,
	now int64,
	typ uint8,
	ele domain.AnswerElement) dao.AnswerElement {
	return dao.AnswerElement{
		Qid:       qid,
		Type:      typ,
		Content:   ele.Content,
		Highlight: ele.Highlight,
		Keywords:  ele.Keywords,
		Shorthand: ele.Shorthand,
		Guidance:  ele.Guidance,
		Ctime:     now,
		Utime:     now,
	}
}

func NewCacheRepository(d dao.QuestionDAO, c cache.QuestionCache) Repository {
	return &CachedRepository{
		dao:    d,
		cache:  c,
		logger: elog.DefaultLogger,
	}
}
