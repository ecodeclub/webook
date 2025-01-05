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

	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/webook/internal/project/internal/event"
	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ecodeclub/webook/internal/project/internal/repository"
)

// Service C 端接口
type Service interface {
	List(ctx context.Context, offset int, limit int) (int64, []domain.Project, error)
	Detail(ctx context.Context, id int64) (domain.Project, error)
	// Brief 获得 project 本身的内容
	Brief(ctx context.Context, id int64) (domain.Project, error)
}

var _ Service = &service{}

type service struct {
	repo     repository.Repository
	producer event.InteractiveEventProducer
	logger   *elog.Component
}

func (s *service) Brief(ctx context.Context, id int64) (domain.Project, error) {
	return s.repo.Brief(ctx, id)
}

func (s *service) Detail(ctx context.Context, id int64) (domain.Project, error) {
	prj, err := s.repo.Detail(ctx, id)
	if err == nil {
		go func() {
			newCtx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			err1 := s.producer.Produce(newCtx, event.NewViewCntEvent(id, domain.BizProject))
			if err1 != nil {
				if err1 != nil {
					s.logger.Error("发送问题阅读计数消息到消息队列失败", elog.FieldErr(err1), elog.Int64("pid", id))
				}
			}
		}()
	}
	return prj, err
}

func (s *service) List(ctx context.Context, offset int, limit int) (int64, []domain.Project, error) {
	var (
		eg       errgroup.Group
		total    int64
		projects []domain.Project
	)
	eg.Go(func() error {
		var err error
		projects, err = s.repo.List(ctx, offset, limit)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.Count(ctx)
		return err
	})
	return total, projects, eg.Wait()
}

func NewService(repo repository.Repository, producer event.InteractiveEventProducer) Service {
	return &service{repo: repo, producer: producer, logger: elog.DefaultLogger}
}
