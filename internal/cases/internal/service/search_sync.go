package service

import (
	"context"
	"fmt"
	"github.com/ecodeclub/webook/internal/cases/internal/event"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"github.com/gotomicro/ego/core/elog"
	"github.com/olivere/elastic/v7"
	"golang.org/x/sync/errgroup"
	"time"
)

const (
	defaultPageSize = 10
	caseIndex       = "case_index"
	pubCaseIndex    = "pub_case_index"
	defaultTimeout  = 10 * time.Minute
)

type SearchSyncService interface {
	SyncAll()
}

type caseSearchSyncService struct {
	repo     repository.CaseRepo
	esClient *elastic.Client
	logger   *elog.Component
}

func NewCaseSearchSyncService(repo repository.CaseRepo, esClient *elastic.Client) SearchSyncService {
	return &caseSearchSyncService{
		repo:     repo,
		esClient: esClient,
	}
}

func (c *caseSearchSyncService) SyncAll() {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	var eg errgroup.Group
	eg.Go(func() error {
		return c.caseSync(ctx)
	})
	eg.Go(func() error {
		return c.pubCaseSync(ctx)
	})
	if err := eg.Wait(); err != nil {
		c.logger.Error("同步失败", elog.FieldErr(err))
	}
}

func (s *caseSearchSyncService) caseSync(ctx context.Context) error {
	offset := 0
	for {
		cases, err := s.repo.ListSync(ctx, offset, defaultPageSize)
		if err != nil {
			return err
		}
		if len(cases) == 0 {
			break
		}
		for _, ca := range cases {
			evt := event.NewCaseEvent(ca)
			_, err = s.esClient.Index().
				Index(caseIndex).
				Id(fmt.Sprintf("%d", ca.Id)).
				BodyString(evt.Data).
				Do(ctx)
			if err != nil {

				return err
			}
		}
		offset += len(cases)
	}
	return nil
}

func (s *caseSearchSyncService) pubCaseSync(ctx context.Context) error {
	offset := 0
	for {
		cases, err := s.repo.PubListSync(ctx, offset, defaultPageSize)
		if err != nil {
			return err
		}
		if len(cases) == 0 {
			break
		}
		for _, ca := range cases {
			evt := event.NewCaseEvent(ca)
			_, err = s.esClient.Index().
				Index(pubCaseIndex).
				Id(fmt.Sprintf("%d", ca.Id)).
				BodyString(evt.Data).
				Do(ctx)
			if err != nil {
				s.logger.Error("", elog.FieldErr(err))
				continue
			}
		}
		// Move to next batch
		offset += len(cases)
	}
	return nil
}
