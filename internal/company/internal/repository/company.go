package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/company/internal/domain"
	"github.com/ecodeclub/webook/internal/company/internal/repository/dao"
)

type CompanyRepository interface {
	Save(ctx context.Context, c domain.Company) (int64, error)
	FindById(ctx context.Context, id int64) (domain.Company, error)
	FindByIds(ctx context.Context, ids []int64) ([]domain.Company, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Company, error)
	Count(ctx context.Context) (int64, error)
	Delete(ctx context.Context, id int64) error
}

type companyRepository struct {
	dao dao.CompanyDAO
}

func NewCompanyRepository(dao dao.CompanyDAO) CompanyRepository {
	return &companyRepository{
		dao: dao,
	}
}

func (r *companyRepository) Save(ctx context.Context, c domain.Company) (int64, error) {
	return r.dao.Save(ctx, r.domainToEntity(c))
}

func (r *companyRepository) FindById(ctx context.Context, id int64) (domain.Company, error) {
	entity, err := r.dao.FindById(ctx, id)
	if err != nil {
		return domain.Company{}, err
	}
	return r.entityToDomain(entity), nil
}

func (r *companyRepository) FindByIds(ctx context.Context, ids []int64) ([]domain.Company, error) {
	entities, err := r.dao.FindByIds(ctx, ids)
	if err != nil {
		return nil, err
	}

	companies := make([]domain.Company, 0, len(entities))
	for _, entity := range entities {
		companies = append(companies, r.entityToDomain(entity))
	}
	return companies, nil
}

func (r *companyRepository) List(ctx context.Context, offset int, limit int) ([]domain.Company, error) {
	entities, err := r.dao.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}

	companies := make([]domain.Company, 0, len(entities))
	for _, entity := range entities {
		companies = append(companies, r.entityToDomain(entity))
	}
	return companies, nil
}

func (r *companyRepository) Count(ctx context.Context) (int64, error) {
	return r.dao.Count(ctx)
}

func (r *companyRepository) Delete(ctx context.Context, id int64) error {
	return r.dao.DeleteById(ctx, id)
}

func (r *companyRepository) domainToEntity(c domain.Company) dao.Company {
	return dao.Company{
		Id:    c.ID,
		Name:  c.Name,
		Ctime: c.Ctime,
		Utime: c.Utime,
	}
}

func (r *companyRepository) entityToDomain(c dao.Company) domain.Company {
	return domain.Company{
		ID:    c.Id,
		Name:  c.Name,
		Ctime: c.Ctime,
		Utime: c.Utime,
	}
}
