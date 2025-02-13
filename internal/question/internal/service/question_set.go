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
	"time"

	"github.com/ecodeclub/ekit/slice"

	"github.com/ecodeclub/webook/internal/question/internal/event"
	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
	"golang.org/x/sync/errgroup"
)

// QuestionSetService 还没有分离制作库和线上库
//
//go:generate mockgen -source=./question_set.go -destination=../../mocks/quetion_set.mock.go -package=quemocks -typed=true QuestionSetService
type QuestionSetService interface {
	Save(ctx context.Context, set domain.QuestionSet) (int64, error)
	UpdateQuestions(ctx context.Context, set domain.QuestionSet) error
	List(ctx context.Context, offset, limit int) ([]domain.QuestionSet, int64, error)
	ListDefault(ctx context.Context, offset, limit int) ([]domain.QuestionSet, int64, error)
	Detail(ctx context.Context, id int64) (domain.QuestionSet, error)
	GetByIds(ctx context.Context, ids []int64) ([]domain.QuestionSet, error)
	DetailByBiz(ctx context.Context, biz string, bizId int64) (domain.QuestionSet, error)
	GetCandidates(ctx context.Context, id int64, offset int, limit int) ([]domain.Question, int64, error)
	GetByIDsWithQuestion(ctx context.Context, ids []int64) ([]domain.QuestionSet, error)

	PubDetail(ctx context.Context, id int64) (domain.QuestionSet, error)
}

type questionSetService struct {
	repo         repository.QuestionSetRepository
	queRepo      repository.Repository
	producer     event.SyncDataToSearchEventProducer
	intrProducer event.InteractiveEventProducer
	logger       *elog.Component
	syncTimeout  time.Duration
}

func (q *questionSetService) GetByIDsWithQuestion(ctx context.Context, ids []int64) ([]domain.QuestionSet, error) {
	return q.repo.GetByIDsWithQuestion(ctx, ids)
}

func (q *questionSetService) GetCandidates(ctx context.Context, id int64, offset int, limit int) ([]domain.Question, int64, error) {
	qs, err := q.repo.GetByID(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	qids := slice.Map(qs.Questions, func(idx int, src domain.Question) int64 {
		return src.Id
	})
	if len(qids) == 0 {
		// 这是一种很 tricky 的写法，可以简化代码
		qids = append(qids, -1)
	}
	return q.queRepo.ExcludeQuestions(ctx, qids, offset, limit)
}

func (q *questionSetService) DetailByBiz(ctx context.Context, biz string, bizId int64) (domain.QuestionSet, error) {
	return q.repo.GetByBiz(ctx, biz, bizId)
}

func (q *questionSetService) ListDefault(ctx context.Context, offset, limit int) ([]domain.QuestionSet, int64, error) {
	var (
		eg    errgroup.Group
		qs    []domain.QuestionSet
		total int64
	)
	eg.Go(func() error {
		var err error
		qs, err = q.repo.ListByBiz(ctx, offset, limit, domain.DefaultBiz)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = q.repo.CountByBiz(ctx, domain.DefaultBiz)
		return err
	})
	return qs, total, eg.Wait()

}

func (q *questionSetService) GetByIds(ctx context.Context, ids []int64) ([]domain.QuestionSet, error) {
	return q.repo.GetByIDs(ctx, ids)
}

func (q *questionSetService) Save(ctx context.Context, set domain.QuestionSet) (int64, error) {
	var id = set.Id
	var err error
	if set.Id > 0 {
		err = q.repo.UpdateNonZero(ctx, set)
	} else {
		id, err = q.repo.Create(ctx, set)
	}
	if err != nil {
		return 0, err
	}
	q.syncQuestionSet(id)
	return id, nil
}

func (q *questionSetService) UpdateQuestions(ctx context.Context, set domain.QuestionSet) error {
	err := q.repo.UpdateQuestions(ctx, set)
	if err != nil {
		return err
	}
	q.syncQuestionSet(set.Id)
	return nil
}

func (q *questionSetService) Detail(ctx context.Context, id int64) (domain.QuestionSet, error) {
	qs, err := q.repo.GetByID(ctx, id)
	if err == nil {
		// 没有区分 B 端还是 C 端，但是这种计数不需要精确计算
		go func() {
			newCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			err1 := q.intrProducer.Produce(newCtx, event.NewViewCntEvent(id, domain.QuestionSetBiz))
			if err1 != nil {
				q.logger.Error("发送阅读计数消息到消息队列失败", elog.FieldErr(err1), elog.Int64("qsid", id))
			}
		}()
	}
	return qs, err
}

func (q *questionSetService) PubDetail(ctx context.Context, id int64) (domain.QuestionSet, error) {
	qs, err := q.repo.PubGetByID(ctx, id)
	if err == nil {
		// 没有区分 B 端还是 C 端，但是这种计数不需要精确计算
		go func() {
			newCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			err1 := q.intrProducer.Produce(newCtx, event.NewViewCntEvent(id, domain.QuestionSetBiz))
			if err1 != nil {
				q.logger.Error("发送阅读计数消息到消息队列失败", elog.FieldErr(err1), elog.Int64("qsid", id))
			}
		}()
	}
	return qs, err
}

func (q *questionSetService) List(ctx context.Context, offset, limit int) ([]domain.QuestionSet, int64, error) {
	var (
		eg    errgroup.Group
		qs    []domain.QuestionSet
		total int64
	)
	eg.Go(func() error {
		var err error
		qs, err = q.repo.List(ctx, offset, limit)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = q.repo.Total(ctx)
		return err
	})
	return qs, total, eg.Wait()
}

func (q *questionSetService) syncQuestionSet(id int64) {
	ctx, cancel := context.WithTimeout(context.Background(), q.syncTimeout)
	defer cancel()
	qSet, err := q.repo.GetByID(ctx, id)
	if err != nil {
		q.logger.Error("发送同步搜索信息",
			elog.FieldErr(err),
		)
		return
	}
	evt := event.NewQuestionSetEvent(qSet)
	err = q.producer.Produce(ctx, evt)
	if err != nil {
		q.logger.Error("发送同步搜索信息",
			elog.FieldErr(err),
			elog.Any("event", evt),
		)
	}
}

func NewQuestionSetService(repo repository.QuestionSetRepository,
	queRepo repository.Repository,
	intrProducer event.InteractiveEventProducer,
	producer event.SyncDataToSearchEventProducer) QuestionSetService {
	return &questionSetService{
		repo:         repo,
		queRepo:      queRepo,
		producer:     producer,
		intrProducer: intrProducer,
		logger:       elog.DefaultLogger,
		syncTimeout:  10 * time.Second,
	}
}
