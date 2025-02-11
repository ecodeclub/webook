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

	"github.com/ecodeclub/webook/internal/question/internal/event"
	"github.com/gotomicro/ego/core/elog"

	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
)

// Service TODO 要分离制作库接口和线上库接口
//
//go:generate mockgen -source=./question.go -destination=../../mocks/question.mock.go -package=quemocks -typed=true Service
type Service interface {
	// Save 保存数据，question 绝对不会为 nil
	Save(ctx context.Context, question *domain.Question) (int64, error)
	Publish(ctx context.Context, que *domain.Question) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Question, int64, error)
	Detail(ctx context.Context, qid int64) (domain.Question, error)
	// Delete 会直接删除制作库和线上库的数据
	Delete(ctx context.Context, qid int64) error

	// PubList 只会返回八股文的数据
	PubList(ctx context.Context, offset int, limit int) (int64, []domain.Question, error)
	// GetPubByIDs 目前只会获取基础信息，也就是不包括答案在内的信息
	GetPubByIDs(ctx context.Context, ids []int64) ([]domain.Question, error)
	PubDetail(ctx context.Context, qid int64) (domain.Question, error)
}

type service struct {
	repo                  repository.Repository
	syncProducer          event.SyncDataToSearchEventProducer
	intrProducer          event.InteractiveEventProducer
	knowledgeBaseProducer event.KnowledgeBaseEventProducer

	logger      *elog.Component
	syncTimeout time.Duration
}

func (s *service) GetPubByIDs(ctx context.Context, ids []int64) ([]domain.Question, error) {
	return s.repo.GetPubByIDs(ctx, ids)
}

func (s *service) PubDetail(ctx context.Context, qid int64) (domain.Question, error) {
	que, err := s.repo.GetPubByID(ctx, qid)
	if err == nil {
		go func() {
			newCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			err1 := s.intrProducer.Produce(newCtx, event.NewViewCntEvent(qid, domain.QuestionBiz))
			if err1 != nil {
				s.logger.Error("发送问题阅读计数消息到消息队列失败", elog.FieldErr(err1), elog.Int64("qid", qid))
			}
		}()
	}
	return que, err
}

func (s *service) Detail(ctx context.Context, qid int64) (domain.Question, error) {
	return s.repo.GetById(ctx, qid)
}

func (s *service) Delete(ctx context.Context, qid int64) error {
	return s.repo.Delete(ctx, qid)
}

func (s *service) List(ctx context.Context, offset int, limit int) ([]domain.Question, int64, error) {
	var (
		eg    errgroup.Group
		qs    []domain.Question
		total int64
	)
	eg.Go(func() error {
		var err error
		qs, err = s.repo.List(ctx, offset, limit)
		return err
	})

	eg.Go(func() error {
		var err error
		total, err = s.repo.Total(ctx)
		return err
	})
	return qs, total, eg.Wait()
}

func (s *service) PubList(ctx context.Context, offset int, limit int) (int64, []domain.Question, error) {
	var (
		eg    errgroup.Group
		qs    []domain.Question
		total int64
	)
	eg.Go(func() error {
		var err error
		qs, err = s.repo.PubList(ctx, offset, limit, domain.DefaultBiz)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.PubCount(ctx, domain.DefaultBiz)
		return err
	})
	return total, qs, eg.Wait()
}

func (s *service) Save(ctx context.Context, question *domain.Question) (int64, error) {
	question.Status = domain.UnPublishedStatus
	var id = question.Id
	var err error
	if question.Id > 0 {
		err = s.repo.Update(ctx, question)
	} else {
		id, err = s.repo.Create(ctx, question)
	}
	if err != nil {
		return 0, err
	}
	qctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	que, eerr := s.getQuestion(ctx, id)
	if eerr != nil {
		return id, nil
	}
	s.syncQuestion(qctx, que)
	return id, nil
}

func (s *service) Publish(ctx context.Context, question *domain.Question) (int64, error) {
	question.Status = domain.PublishedStatus
	id, err := s.repo.Sync(ctx, question)
	if err != nil {
		return 0, err
	}
	// 获取问题
	qctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	que, eerr := s.getQuestion(qctx, id)
	if eerr != nil {
		return id, nil
	}
	// 同步到搜索服务
	s.syncQuestion(qctx, que)
	// 上传到知识库
	s.uploadQuestion(qctx, que)
	return id, nil
}

func NewService(repo repository.Repository,
	syncEvent event.SyncDataToSearchEventProducer,
	intrEvent event.InteractiveEventProducer,
	knowledgeBaseProducer event.KnowledgeBaseEventProducer,
) Service {
	return &service{
		repo:                  repo,
		syncProducer:          syncEvent,
		intrProducer:          intrEvent,
		knowledgeBaseProducer: knowledgeBaseProducer,
		logger:                elog.DefaultLogger,
		syncTimeout:           10 * time.Second,
	}
}

func (s *service) syncQuestion(ctx context.Context, que domain.Question) {
	evt := event.NewQuestionEvent(que)
	err := s.syncProducer.Produce(ctx, evt)
	if err != nil {
		s.logger.Error("发送同步搜索信息",
			elog.FieldErr(err),
			elog.Any("event", evt),
		)
	}
}

func (s *service) getQuestion(ctx context.Context, id int64) (domain.Question, error) {
	que, err := s.repo.GetById(ctx, id)
	if err != nil {
		s.logger.Error("发送同步搜索信息",
			elog.FieldErr(err),
		)
		return domain.Question{}, err
	}
	return que, nil
}

func (s *service) uploadQuestion(ctx context.Context, que domain.Question) {
	evt, err := event.NewKnowledgeBaseEvent(que)
	if err != nil {
		s.logger.Error("获取知识库事件失败",
			elog.FieldErr(err),
		)
		return
	}
	err = s.knowledgeBaseProducer.Produce(ctx, evt)
	if err != nil {
		s.logger.Error("发送上传知识库事件失败",
			elog.FieldErr(err),
			elog.Any("event", evt),
		)
	}
}
