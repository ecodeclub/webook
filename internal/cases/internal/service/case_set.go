package service

import (
	"context"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"golang.org/x/sync/errgroup"
)

type CaseSetService interface {
	Save(ctx context.Context, set domain.CaseSet) (int64, error)
	UpdateCases(ctx context.Context, set domain.CaseSet) error
	List(ctx context.Context, offset, limit int) ([]domain.CaseSet, int64, error)
	Detail(ctx context.Context, id int64) (domain.CaseSet, error)
	GetByIds(ctx context.Context, ids []int64) ([]domain.CaseSet, error)

	ListByBiz(ctx context.Context, offset, limit int, biz string) ([]domain.CaseSet, error)
	ListDefault(ctx context.Context, offset, limit int) ([]domain.CaseSet, error)
	GetByBiz(ctx context.Context, biz string, bizId int64) (domain.CaseSet, error)
}


type casSetSvc struct {
	repo repository.CaseSetRepository
}

func NewCaseSetService(repo repository.CaseSetRepository) CaseSetService {
	return &casSetSvc{repo: repo}
}

func (c *casSetSvc) ListDefault(ctx context.Context, offset, limit int) ([]domain.CaseSet, error) {
	return c.repo.ListByBiz(ctx, offset, limit, domain.DefaultBiz)
}

func (c *casSetSvc) ListByBiz(ctx context.Context, offset, limit int, biz string) ([]domain.CaseSet, error) {
	return c.repo.ListByBiz(ctx, offset, limit, biz)
}

func (c *casSetSvc) GetByBiz(ctx context.Context, biz string, bizId int64) (domain.CaseSet, error) {
	return c.repo.GetByBiz(ctx, biz, bizId)
}

func (c *casSetSvc) Save(ctx context.Context, set domain.CaseSet) (int64, error) {
	var id =set.ID
	var err error
	if set.ID > 0 {
		err  = c.repo.UpdateNonZero(ctx, set)
	}else {
		id ,err = c.repo.Create(ctx, set)
	}
	return id, err
}

func (c *casSetSvc) UpdateCases(ctx context.Context, set domain.CaseSet) error {
	return c.repo.UpdateCases(ctx, set)
}

func (c *casSetSvc) List(ctx context.Context, offset, limit int) ([]domain.CaseSet, int64, error) {
	var eg errgroup.Group
	var sets []domain.CaseSet
	var total int64
	eg.Go(func() error {
		var eerr error
		sets,eerr = c.repo.List(ctx, offset, limit)
		return eerr
	})
	eg.Go(func() error {
		var eerr error
		total,eerr = c.repo.Total(ctx)
		return eerr
	})

	if err := eg.Wait();err != nil {
		return nil, 0, err
	}
	return sets, total, nil
}

func (c *casSetSvc) Detail(ctx context.Context, id int64) (domain.CaseSet, error) {
	return c.repo.GetByID(ctx, id)
}

func (c *casSetSvc) GetByIds(ctx context.Context, ids []int64) ([]domain.CaseSet, error) {
	return c.repo.GetByIDs(ctx, ids)
}

