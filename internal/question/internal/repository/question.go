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

	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/ekit/sqlx"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/question/internal/repository/cache"
	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
)

const (
	cacheMax = 50
	cacheMin = 0
)

type Repository interface {
	PubList(ctx context.Context, offset int, limit int, biz string) ([]domain.Question, error)
	// Sync 保存到制作库，而后同步到线上库
	Sync(ctx context.Context, que *domain.Question) (int64, error)
	PubCount(ctx context.Context, biz string) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Question, error)
	Total(ctx context.Context) (int64, error)
	Update(ctx context.Context, question *domain.Question) error
	Create(ctx context.Context, question *domain.Question) (int64, error)
	// QuestionIds 获取全量问题id列表，用于ai同步
	QuestionIds(ctx context.Context) ([]int64, error)

	// Delete 会直接删除制作库和线上库的数据
	Delete(ctx context.Context, qid int64) error

	GetById(ctx context.Context, qid int64) (domain.Question, error)
	GetPubByID(ctx context.Context, qid int64) (domain.Question, error)
	GetPubByIDs(ctx context.Context, ids []int64) ([]domain.Question, error)
	// ExcludeQuestions 分页接口，不含这些 id 的问题
	ExcludeQuestions(ctx context.Context, ids []int64, offset int, limit int) ([]domain.Question, int64, error)
}

// CachedRepository 支持缓存的 repository 实现
// 虽然在 Save 的时候，理论上也需要更新缓存，但是没有必要
type CachedRepository struct {
	dao    dao.QuestionDAO
	cache  cache.QuestionCache
	logger *elog.Component
}

func (c *CachedRepository) PubCount(ctx context.Context, biz string) (int64, error) {
	total, cacheErr := c.cache.GetTotal(ctx, biz)
	if cacheErr == nil {
		return total, nil
	}
	total, err := c.dao.PubCount(ctx, biz)
	if err != nil {
		return 0, err
	}
	cacheErr = c.cache.SetTotal(ctx, biz, total)
	if cacheErr != nil {
		// 记录一下日志
		c.logger.Error("记录缓存失败", elog.FieldErr(cacheErr))
	}
	return total, nil
}

func (c *CachedRepository) QuestionIds(ctx context.Context) ([]int64, error) {
	return c.dao.Ids(ctx)
}

func (c *CachedRepository) GetPubByIDs(ctx context.Context, qids []int64) ([]domain.Question, error) {
	data, err := c.dao.GetPubByIDs(ctx, qids)
	return slice.Map(data, func(idx int, src dao.PublishQuestion) domain.Question {
		return c.toDomain(dao.Question(src))
	}), err
}

func (c *CachedRepository) GetPubByID(ctx context.Context, qid int64) (domain.Question, error) {
	// 可以缓存
	question, cacheErr := c.cache.GetQuestion(ctx, qid)
	// 找到直接返回
	if cacheErr == nil {
		return question, nil
	}

	entityQuestion, err := c.getPubByIDFromDb(ctx, qid)
	if err != nil {
		return domain.Question{}, err
	}
	cacheErr = c.cache.SetQuestion(ctx, entityQuestion)
	if cacheErr != nil {
		// 记录一下日志
		c.logger.Error("记录缓存失败", elog.FieldErr(cacheErr))
	}
	return entityQuestion, nil
}

func (c *CachedRepository) getPubByIDFromDb(ctx context.Context, qid int64) (domain.Question, error) {
	data, pubEles, err := c.dao.GetPubByID(ctx, qid)
	if err != nil {
		return domain.Question{}, err
	}
	eles := slice.Map(pubEles, func(idx int, src dao.PublishAnswerElement) dao.AnswerElement {
		return dao.AnswerElement(src)
	})
	entityQuestion := c.toDomainWithAnswer(dao.Question(data), eles)
	return entityQuestion, nil
}

func (c *CachedRepository) ExcludeQuestions(ctx context.Context, ids []int64, offset int, limit int) ([]domain.Question, int64, error) {
	var (
		eg   errgroup.Group
		cnt  int64
		data []dao.Question
	)
	eg.Go(func() error {
		var err error
		cnt, err = c.dao.NotInTotal(ctx, ids)
		return err
	})

	eg.Go(func() error {
		var err error
		data, err = c.dao.NotIn(ctx, ids, offset, limit)
		return err
	})
	err := eg.Wait()
	return slice.Map(data, func(idx int, src dao.Question) domain.Question {
		return c.toDomain(src)
	}), cnt, err
}

func (c *CachedRepository) GetById(ctx context.Context, qid int64) (domain.Question, error) {
	data, eles, err := c.dao.GetByID(ctx, qid)
	if err != nil {
		return domain.Question{}, err
	}
	return c.toDomainWithAnswer(data, eles), nil
}

func (c *CachedRepository) Delete(ctx context.Context, qid int64) error {
	que, _, err := c.dao.GetByID(ctx, qid)
	if err != nil {
		return nil
	}
	err = c.dao.Delete(ctx, qid)
	if err != nil {
		return err
	}
	//
	cacheErr := c.cache.DelQuestion(ctx, qid)
	if cacheErr != nil {
		// 记录一下日志
		c.logger.Error("删除题目缓存失败", elog.FieldErr(cacheErr), elog.Int64("qid", qid))
	}
	cacheErr = c.cacheList(ctx, que.Biz)
	if cacheErr != nil {
		// 记录一下日志
		c.logger.Error("设置题目列表缓存失败", elog.FieldErr(cacheErr), elog.String("biz", que.Biz))
	}
	return nil
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
	id, err := c.dao.Sync(ctx, q, eles)
	if err != nil {
		return id, err
	}
	// todo 以后重构，现直接从数据库中获取，写入缓存
	questionEntity, cacheErr := c.getPubByIDFromDb(ctx, id)
	if cacheErr != nil {
		// 记录一下日志
		c.logger.Error("设置题目缓存失败", elog.FieldErr(cacheErr), elog.Int64("qid", id))
	}
	cacheErr = c.cache.SetQuestion(ctx, questionEntity)
	if cacheErr != nil {
		// 记录一下日志
		c.logger.Error("设置题目缓存失败", elog.FieldErr(cacheErr), elog.Int64("qid", id))
	}

	// 更新前50条的缓存
	cacheErr = c.cacheList(ctx, que.Biz)
	if cacheErr != nil {
		// 记录一下日志
		c.logger.Error("设置题目列表缓存失败", elog.FieldErr(cacheErr), elog.String("biz", que.Biz))
	}
	// 更新总数
	cacheErr = c.cacheTotal(ctx, que.Biz)
	if cacheErr != nil {
		// 记录一下日志
		c.logger.Error("设置题目总数缓存失败", elog.FieldErr(cacheErr), elog.String("biz", que.Biz))
	}
	return id, nil
}

func (c *CachedRepository) cacheList(ctx context.Context, biz string) error {
	list, err := c.dao.PubList(ctx, cacheMin, cacheMax, biz)
	if err != nil {
		return err
	}
	qs := slice.Map(list, func(idx int, src dao.PublishQuestion) domain.Question {
		return c.toDomain(dao.Question(src))
	})
	return c.cache.SetQuestions(ctx, biz, qs)
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

func (c *CachedRepository) PubList(ctx context.Context, offset int, limit int, biz string) ([]domain.Question, error) {
	// TODO 缓存第一页
	if c.checkTop50(offset, limit) {
		// 可以从缓存获取
		qs, err := c.cache.GetQuestions(ctx, biz)
		if err == nil {
			return c.getQuestionsFromCache(qs, offset, limit), nil
		}
		// 未命中缓存
		daoqs, err := c.dao.PubList(ctx, cacheMin, cacheMax, biz)
		if err != nil {
			return nil, err
		}
		qs = slice.Map(daoqs, func(idx int, src dao.PublishQuestion) domain.Question {
			return c.toDomain(dao.Question(src))
		})

		cacheErr := c.cache.SetQuestions(ctx, biz, qs)
		if cacheErr != nil {
			c.logger.Error("设置题目列表缓存失败", elog.FieldErr(cacheErr), elog.String("biz", biz))
		}
		return c.getQuestionsFromCache(qs, offset, limit), nil
	}

	qs, err := c.dao.PubList(ctx, offset, limit, biz)
	return slice.Map(qs, func(idx int, src dao.PublishQuestion) domain.Question {
		return c.toDomain(dao.Question(src))
	}), err
}

// 校验数据是否都存在于缓存中
func (c *CachedRepository) checkTop50(offset, limit int) bool {
	last := offset + limit
	return last <= cacheMax
}

func (c *CachedRepository) getQuestionsFromCache(questions []domain.Question, offset, limit int) []domain.Question {
	if offset >= len(questions) {
		return []domain.Question{}
	}
	remain := len(questions) - offset
	if remain > limit {
		remain = limit
	}
	res := make([]domain.Question, 0, remain)
	for i := offset; i < offset+remain; i++ {
		res = append(res, questions[i])
	}
	return res
}

func (c *CachedRepository) cacheTotal(ctx context.Context, biz string) error {
	count, err := c.dao.PubCount(ctx, biz)
	if err != nil {
		return err
	}
	return c.cache.SetTotal(ctx, biz, count)
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
		Biz:     que.Biz,
		BizId:   que.BizId,
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
		Biz:     que.Biz,
		BizId:   que.BizId,
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
