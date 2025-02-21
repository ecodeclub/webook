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

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
	"github.com/gotomicro/ego/core/elog"
)

type QuestionSetRepository interface {
	Create(ctx context.Context, set domain.QuestionSet) (int64, error)
	UpdateQuestions(ctx context.Context, set domain.QuestionSet) error

	GetByID(ctx context.Context, id int64) (domain.QuestionSet, error)
	PubGetByID(ctx context.Context, id int64) (domain.QuestionSet, error)

	Total(ctx context.Context) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.QuestionSet, error)
	UpdateNonZero(ctx context.Context, set domain.QuestionSet) error
	GetByIDs(ctx context.Context, ids []int64) ([]domain.QuestionSet, error)
	GetByIDsWithQuestion(ctx context.Context, ids []int64) ([]domain.QuestionSet, error)
	ListByBiz(ctx context.Context, offset, limit int, biz string) ([]domain.QuestionSet, error)
	GetByBiz(ctx context.Context, biz string, bizId int64) (domain.QuestionSet, error)
	CountByBiz(ctx context.Context, biz string) (int64, error)
}

var _ QuestionSetRepository = &questionSetRepository{}

type questionSetRepository struct {
	dao    dao.QuestionSetDAO
	logger *elog.Component
}

func (q *questionSetRepository) CountByBiz(ctx context.Context, biz string) (int64, error) {
	return q.dao.CountByBiz(ctx, biz)
}
func (q *questionSetRepository) GetByIDsWithQuestion(ctx context.Context, ids []int64) ([]domain.QuestionSet, error) {
	qsets, questionMap, err := q.dao.GetByIDsWithQuestions(ctx, ids)
	if err != nil {
		return nil, err
	}
	res := slice.Map(qsets, func(idx int, src dao.QuestionSet) domain.QuestionSet {
		qids := questionMap[src.Id]
		set := domain.QuestionSet{
			Id:    src.Id,
			Title: src.Title,
		}
		questions := slice.Map(qids, func(idx int, src dao.Question) domain.Question {
			return domain.Question{
				Id:    src.Id,
				Title: src.Title,
			}
		})
		set.Questions = questions
		return set
	})
	return res, nil
}

func (q *questionSetRepository) GetByBiz(ctx context.Context, biz string, bizId int64) (domain.QuestionSet, error) {
	set, err := q.dao.GetByBiz(ctx, biz, bizId)
	if err != nil {
		return domain.QuestionSet{}, err
	}
	questions, err := q.getDomainQuestions(ctx, set.Id)
	if err != nil {
		return domain.QuestionSet{}, err
	}
	return domain.QuestionSet{
		Id:          set.Id,
		Uid:         set.Uid,
		Title:       set.Title,
		Biz:         set.Biz,
		BizId:       set.BizId,
		Description: set.Description,
		Questions:   questions,
		Utime:       time.UnixMilli(set.Utime),
	}, nil
}

func (q *questionSetRepository) ListByBiz(ctx context.Context, offset, limit int, biz string) ([]domain.QuestionSet, error) {
	qs, err := q.dao.ListByBiz(ctx, offset, limit, biz)
	if err != nil {
		return nil, err
	}
	return slice.Map(qs, func(idx int, src dao.QuestionSet) domain.QuestionSet {
		return q.toDomainQuestionSet(src)
	}), err
}

func (q *questionSetRepository) GetByIDs(ctx context.Context, ids []int64) ([]domain.QuestionSet, error) {
	qs, err := q.dao.GetByIDs(ctx, ids)
	return slice.Map(qs, func(idx int, src dao.QuestionSet) domain.QuestionSet {
		return q.toDomainQuestionSet(src)
	}), err
}

func (q *questionSetRepository) UpdateNonZero(ctx context.Context, set domain.QuestionSet) error {
	return q.dao.UpdateNonZero(ctx, q.toEntityQuestionSet(set))
}

func (q *questionSetRepository) Create(ctx context.Context, set domain.QuestionSet) (int64, error) {
	return q.dao.Create(ctx, q.toEntityQuestionSet(set))
}

func (q *questionSetRepository) toEntityQuestionSet(d domain.QuestionSet) dao.QuestionSet {
	return dao.QuestionSet{
		Id:          d.Id,
		Uid:         d.Uid,
		Title:       d.Title,
		Biz:         d.Biz,
		BizId:       d.BizId,
		Description: d.Description,
		Utime:       d.Utime.UnixMilli(),
	}
}

func (q *questionSetRepository) UpdateQuestions(ctx context.Context, set domain.QuestionSet) error {
	qids := make([]int64, len(set.Questions))
	for i := range set.Questions {
		qids[i] = set.Questions[i].Id
	}
	return q.dao.UpdateQuestionsByID(ctx, set.Id, qids)
}

func (q *questionSetRepository) getPubDomainQuestions(ctx context.Context, id int64) ([]domain.Question, error) {
	questions, err := q.dao.GetPubQuestionsByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return slice.Map(questions, func(idx int, src dao.PublishQuestion) domain.Question {
		return q.toDomainQuestion(dao.Question(src))
	}), err
}

func (q *questionSetRepository) getDomainQuestions(ctx context.Context, id int64) ([]domain.Question, error) {
	questions, err := q.dao.GetQuestionsByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return slice.Map(questions, func(idx int, src dao.Question) domain.Question {
		return q.toDomainQuestion(src)
	}), err
}

func (q *questionSetRepository) toDomainQuestion(que dao.Question) domain.Question {
	return domain.Question{
		Id:      que.Id,
		Uid:     que.Uid,
		Title:   que.Title,
		Labels:  que.Labels.Val,
		Content: que.Content,
		Biz:     que.Biz,
		BizId:   que.BizId,
		Answer:  domain.Answer{},
		Utime:   time.UnixMilli(que.Utime),
	}
}

func (q *questionSetRepository) GetByID(ctx context.Context, id int64) (domain.QuestionSet, error) {
	set, err := q.dao.GetByID(ctx, id)
	if err != nil {
		return domain.QuestionSet{}, err
	}
	questions, err := q.getDomainQuestions(ctx, id)
	if err != nil {
		return domain.QuestionSet{}, err
	}

	return domain.QuestionSet{
		Id:          set.Id,
		Uid:         set.Uid,
		Title:       set.Title,
		Biz:         set.Biz,
		BizId:       set.BizId,
		Description: set.Description,
		Questions:   questions,
		Utime:       time.UnixMilli(set.Utime),
	}, nil
}

func (q *questionSetRepository) PubGetByID(ctx context.Context, id int64) (domain.QuestionSet, error) {
	set, err := q.dao.GetByID(ctx, id)
	if err != nil {
		return domain.QuestionSet{}, err
	}
	questions, err := q.getPubDomainQuestions(ctx, id)
	if err != nil {
		return domain.QuestionSet{}, err
	}

	return domain.QuestionSet{
		Id:          set.Id,
		Uid:         set.Uid,
		Title:       set.Title,
		Biz:         set.Biz,
		BizId:       set.BizId,
		Description: set.Description,
		Questions:   questions,
		Utime:       time.UnixMilli(set.Utime),
	}, nil
}

func (q *questionSetRepository) Total(ctx context.Context) (int64, error) {
	return q.dao.Count(ctx)
}

func (q *questionSetRepository) List(ctx context.Context, offset int, limit int) ([]domain.QuestionSet, error) {
	qs, err := q.dao.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map(qs, func(idx int, src dao.QuestionSet) domain.QuestionSet {
		return q.toDomainQuestionSet(src)
	}), err
}

func (q *questionSetRepository) toDomainQuestionSet(qs dao.QuestionSet) domain.QuestionSet {
	return domain.QuestionSet{
		Id:          qs.Id,
		Uid:         qs.Uid,
		Title:       qs.Title,
		Biz:         qs.Biz,
		BizId:       qs.BizId,
		Description: qs.Description,
		// Questions:   q.getDomainQuestions(),
		Utime: time.UnixMilli(qs.Utime),
	}
}

func NewQuestionSetRepository(d dao.QuestionSetDAO) QuestionSetRepository {
	return &questionSetRepository{
		dao:    d,
		logger: elog.DefaultLogger}
}
