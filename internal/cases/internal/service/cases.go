package service

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/cases/internal/event"
	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -source=./cases.go -destination=../../mocks/cases.mock.go -package=casemocks -typed Service
type Service interface {
	// Save 保存数据，case 绝对不会为 nil
	Save(ctx context.Context, ca *domain.Case) (int64, error)
	Publish(ctx context.Context, ca *domain.Case) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Case, int64, error)

	PubList(ctx context.Context, offset int, limit int) ([]domain.Case, int64, error)
	GetPubByIDs(ctx context.Context, ids []int64) ([]domain.Case, error)
	Detail(ctx context.Context, caseId int64) (domain.Case, error)
	PubDetail(ctx context.Context, caseId int64) (domain.Case, error)
}

type service struct {
	repo        repository.CaseRepo
	producer    event.SyncEventProducer
	logger      *elog.Component
	syncTimeout time.Duration
}

func (s *service) GetPubByIDs(ctx context.Context, ids []int64) ([]domain.Case, error) {
	return s.repo.GetPubByIDs(ctx, ids)
}

func (s *service) Save(ctx context.Context, ca *domain.Case) (int64, error) {
	ca.Status = domain.UnPublishedStatus
	var id = ca.Id
	var err error
	if ca.Id > 0 {
		err = s.repo.Update(ctx, ca)
	} else {
		id, err = s.repo.Create(ctx, ca)
	}
	if err != nil {
		return 0, err
	}
	s.syncCase(id)
	return id, nil
}

func (s *service) Publish(ctx context.Context, ca *domain.Case) (int64, error) {
	ca.Status = domain.PublishedStatus
	id, err := s.repo.Sync(ctx, ca)
	if err != nil {
		return 0, err
	}
	s.syncCase(id)
	return id, nil
}

func (s *service) List(ctx context.Context, offset int, limit int) ([]domain.Case, int64, error) {
	var (
		total    int64
		caseList []domain.Case
		eg       errgroup.Group
	)
	eg.Go(func() error {
		var err error
		caseList, err = s.repo.List(ctx, offset, limit)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.Total(ctx)
		return err
	})
	if err := eg.Wait(); err != nil {
		return caseList, total, err
	}
	return caseList, total, nil
}

func (s *service) PubList(ctx context.Context, offset int, limit int) ([]domain.Case, int64, error) {

	var (
		total    int64
		caseList []domain.Case
		eg       errgroup.Group
	)
	eg.Go(func() error {
		var err error
		caseList, err = s.repo.PubList(ctx, offset, limit)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = s.repo.PubTotal(ctx)
		return err
	})
	err := eg.Wait()
	return caseList, total, err
}

func (s *service) Detail(ctx context.Context, caseId int64) (domain.Case, error) {
	return s.repo.GetById(ctx, caseId)
}

func (s *service) PubDetail(ctx context.Context, caseId int64) (domain.Case, error) {
	return s.repo.GetPubByID(ctx, caseId)
}

func NewService(repo repository.CaseRepo, producer event.SyncEventProducer) Service {
	return &service{
		repo:        repo,
		producer:    producer,
		logger:      elog.DefaultLogger,
		syncTimeout: 10 * time.Second,
	}
}

func (s *service) syncCase(id int64) {
	ctx, cancel := context.WithTimeout(context.Background(), s.syncTimeout)
	defer cancel()
	ca, err := s.repo.GetById(ctx, id)
	if err != nil {
		s.logger.Error("发送同步搜索信息",
			elog.FieldErr(err),
		)
		return
	}
	evt := event.NewCaseEvent(&ca)
	err = s.producer.Produce(ctx, evt)
	if err != nil {
		s.logger.Error("发送同步搜索信息",
			elog.FieldErr(err),
			elog.Any("event", evt),
		)
	}
}
