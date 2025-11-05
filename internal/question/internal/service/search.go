package service

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v9"

	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/webook/internal/question/internal/event"
	"github.com/ecodeclub/webook/internal/question/internal/repository"
)

const (
	defaultPageSize  = 10
	questionIndex    = "question_index"
	pubQuestionIndex = "pub_question_index"
	defaultTimeout   = 10 * time.Minute // 默认十分钟
)

type SearchSyncService interface {
	SyncAll()
}
type searchSyncService struct {
	repo   repository.Repository
	client *elasticsearch.TypedClient
	logger *elog.Component
}

func NewSearchSyncService(repo repository.Repository, client *elasticsearch.TypedClient) SearchSyncService {
	return &searchSyncService{
		repo:   repo,
		client: client,
		logger: elog.DefaultLogger,
	}
}

func (s *searchSyncService) SyncAll() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error {
		return s.questionSync(ctx)
	})
	eg.Go(func() error {
		return s.pubQuestionSync(ctx)
	})
	if err := eg.Wait(); err != nil {
		s.logger.Error("同步失败", elog.FieldErr(err))
	}
}

func (s *searchSyncService) questionSync(ctx context.Context) error {
	offset := 0
	for {
		questions, err := s.repo.ListSync(ctx, offset, defaultPageSize)
		if err != nil {
			return err
		}
		if len(questions) == 0 {
			break
		}
		for _, q := range questions {
			evt := event.NewQuestionEvent(q)
			_, err = s.client.Index(questionIndex).
				Id(fmt.Sprintf("%d", q.Id)).
				Raw(bytes.NewReader([]byte(evt.Data))).
				Do(ctx)
			if err != nil {
				s.logger.Error("", elog.FieldErr(err))
				continue
			}
		}
		offset += len(questions)
	}
	return nil
}

func (s *searchSyncService) pubQuestionSync(ctx context.Context) error {
	offset := 0
	for {
		questions, err := s.repo.PubListSync(ctx, offset, defaultPageSize)
		if err != nil {
			return err
		}
		if len(questions) == 0 {
			break
		}
		for _, q := range questions {
			evt := event.NewQuestionEvent(q)
			_, err = s.client.Index(pubQuestionIndex).
				Id(fmt.Sprintf("%d", q.Id)).
				Raw(bytes.NewReader([]byte(evt.Data))).
				Do(ctx)
			if err != nil {
				s.logger.Error("", elog.FieldErr(err))
				continue
			}
		}
		// Move to next batch
		offset += len(questions)
	}
	return nil
}
