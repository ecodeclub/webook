package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/company/internal/domain"
	"github.com/ecodeclub/webook/internal/company/internal/repository"
)

type CompanyService interface {
	Save(ctx context.Context, company domain.Company) (int64, error)
	GetById(ctx context.Context, id int64) (domain.Company, error)
	GetByIds(ctx context.Context, ids []int64) ([]domain.Company, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Company, int64, error)
	Delete(ctx context.Context, id int64) error
}

type companyService struct {
	repo repository.CompanyRepository
}

func NewCompanyService(repo repository.CompanyRepository) CompanyService {
	return &companyService{
		repo: repo,
	}
}

func (s *companyService) Save(ctx context.Context, company domain.Company) (int64, error) {
	return s.repo.Save(ctx, company)
}

func (s *companyService) GetById(ctx context.Context, id int64) (domain.Company, error) {
	return s.repo.FindById(ctx, id)
}

func (s *companyService) GetByIds(ctx context.Context, ids []int64) ([]domain.Company, error) {
	return s.repo.FindByIds(ctx, ids)
}

func (s *companyService) List(ctx context.Context, offset int, limit int) ([]domain.Company, int64, error) {
	companies, err := s.repo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return companies, total, nil
}

func (s *companyService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
