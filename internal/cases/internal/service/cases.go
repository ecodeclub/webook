package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"golang.org/x/sync/errgroup"
)

type Service interface {
	// Save 保存数据，case 绝对不会为 nil
	Save(ctx context.Context, ca *domain.Case) (int64, error)
	Publish(ctx context.Context, ca *domain.Case) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Case, int64, error)

	PubList(ctx context.Context, offset int, limit int) ([]domain.Case, int64, error)
	Detail(ctx context.Context, caseId int64) (domain.Case, error)
	PubDetail(ctx context.Context, caseId int64) (domain.Case, error)
}

type service struct {
	repo repository.CaseRepo
}

func NewService(repo repository.CaseRepo) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) Save(ctx context.Context, ca *domain.Case) (int64, error) {
	if ca.Id > 0 {
		return ca.Id, s.repo.Update(ctx, ca)
	}
	return s.repo.Create(ctx, ca)
}

func (s *service) Publish(ctx context.Context, ca *domain.Case) (int64, error) {
	return s.repo.Sync(ctx, ca)
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
	if err := eg.Wait(); err != nil {
		return caseList, total, err
	}
	return caseList, total, nil
}

func (s *service) Detail(ctx context.Context, caseId int64) (domain.Case, error) {
	return s.repo.GetById(ctx, caseId)
}

func (s *service) PubDetail(ctx context.Context, caseId int64) (domain.Case, error) {
	return s.repo.GetPubByID(ctx, caseId)
}
