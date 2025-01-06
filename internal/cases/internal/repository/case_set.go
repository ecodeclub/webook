package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/ekit/mapx"
	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
)

type CaseSetRepository interface {
	CreateCaseSet(ctx context.Context, set domain.CaseSet) (int64, error)
	UpdateCases(ctx context.Context, set domain.CaseSet) error
	GetByID(ctx context.Context, id int64) (domain.CaseSet, error)
	Total(ctx context.Context) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.CaseSet, error)
	UpdateNonZero(ctx context.Context, set domain.CaseSet) error
	CountByBiz(ctx context.Context, biz string) (int64, error)
	GetByIDs(ctx context.Context, ids []int64) ([]domain.CaseSet, error)
	// GetByIDsWithCases 会同步把关联的 Case 也找出来，但是只是找 id，具体内容没有找
	GetByIDsWithCases(ctx context.Context, ids []int64) ([]domain.CaseSet, error)

	ListByBiz(ctx context.Context, offset, limit int, biz string) ([]domain.CaseSet, error)
	GetByBiz(ctx context.Context, biz string, bizId int64) (domain.CaseSet, error)
}

type caseSetRepo struct {
	dao dao.CaseSetDAO
}

func (c *caseSetRepo) CountByBiz(ctx context.Context, biz string) (int64, error) {
	return c.dao.CountByBiz(ctx, biz)
}

func NewCaseSetRepo(caseSetDao dao.CaseSetDAO) CaseSetRepository {
	return &caseSetRepo{
		dao: caseSetDao,
	}
}

func (c *caseSetRepo) ListByBiz(ctx context.Context, offset, limit int, biz string) ([]domain.CaseSet, error) {
	qs, err := c.dao.ListByBiz(ctx, offset, limit, biz)
	if err != nil {
		return nil, err
	}
	return slice.Map(qs, func(idx int, src dao.CaseSet) domain.CaseSet {
		return c.toDomainCaseSet(src)
	}), err
}

func (c *caseSetRepo) GetByBiz(ctx context.Context, biz string, bizId int64) (domain.CaseSet, error) {
	set, err := c.dao.GetByBiz(ctx, biz, bizId)
	if err != nil {
		return domain.CaseSet{}, err
	}
	cases, err := c.getDomainCases(ctx, set.Id)
	if err != nil {
		return domain.CaseSet{}, err
	}
	return domain.CaseSet{
		ID:          set.Id,
		Uid:         set.Uid,
		Title:       set.Title,
		Biz:         set.Biz,
		BizId:       set.BizId,
		Description: set.Description,
		Cases:       cases,
		Utime:       set.Utime,
	}, nil
}

func (c *caseSetRepo) CreateCaseSet(ctx context.Context, set domain.CaseSet) (int64, error) {
	return c.dao.Create(ctx, c.toEntityQuestionSet(set))
}

func (c *caseSetRepo) UpdateCases(ctx context.Context, set domain.CaseSet) error {
	cids := make([]int64, 0, len(set.Cases))
	for i := range set.Cases {
		cids = append(cids, set.Cases[i].Id)
	}
	return c.dao.UpdateCasesByID(ctx, set.ID, cids)
}

func (c *caseSetRepo) GetByID(ctx context.Context, id int64) (domain.CaseSet, error) {
	set, err := c.dao.GetByID(ctx, id)
	if err != nil {
		return domain.CaseSet{}, err
	}
	cases, err := c.getDomainCases(ctx, id)
	if err != nil {
		return domain.CaseSet{}, err
	}

	return domain.CaseSet{
		ID:          set.Id,
		Uid:         set.Uid,
		Title:       set.Title,
		Description: set.Description,
		BizId:       set.BizId,
		Biz:         set.Biz,
		Cases:       cases,
		Utime:       set.Utime,
	}, nil
}

func (c *caseSetRepo) getDomainCases(ctx context.Context, id int64) ([]domain.Case, error) {
	cases, err := c.dao.GetCasesByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return slice.Map(cases, func(idx int, src dao.Case) domain.Case {
		return c.toDomainCase(src)
	}), err
}

func (c *caseSetRepo) Total(ctx context.Context) (int64, error) {
	return c.dao.Count(ctx)
}

func (c *caseSetRepo) List(ctx context.Context, offset int, limit int) ([]domain.CaseSet, error) {
	qs, err := c.dao.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map(qs, func(idx int, src dao.CaseSet) domain.CaseSet {
		return c.toDomainCaseSet(src)
	}), err
}

func (c *caseSetRepo) UpdateNonZero(ctx context.Context, set domain.CaseSet) error {
	return c.dao.UpdateNonZero(ctx, c.toEntityQuestionSet(set))
}

func (c *caseSetRepo) GetByIDs(ctx context.Context, ids []int64) ([]domain.CaseSet, error) {
	qs, err := c.dao.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return slice.Map(qs, func(idx int, src dao.CaseSet) domain.CaseSet {
		return c.toDomainCaseSet(src)
	}), err
}

func (c *caseSetRepo) GetByIDsWithCases(ctx context.Context, ids []int64) ([]domain.CaseSet, error) {
	var (
		eg          errgroup.Group
		qs          []dao.CaseSet
		setCasesMap = mapx.NewMultiBuiltinMap[int64, int64](len(ids))
	)

	eg.Go(func() error {
		var err error
		qs, err = c.dao.GetByIDs(ctx, ids)
		return err
	})

	eg.Go(func() error {
		refs, err := c.dao.GetRefCasesByIDs(ctx, ids)
		for _, ref := range refs {
			_ = setCasesMap.Put(ref.CSID, ref.CID)
		}
		return err
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return slice.Map(qs, func(idx int, src dao.CaseSet) domain.CaseSet {
		cs := c.toDomainCaseSet(src)
		cases, _ := setCasesMap.Get(cs.ID)
		cs.Cases = slice.Map(cases, func(idx int, src int64) domain.Case {
			return domain.Case{Id: src}
		})
		return cs
	}), nil
}

func (c *caseSetRepo) toEntityQuestionSet(d domain.CaseSet) dao.CaseSet {
	return dao.CaseSet{
		Id:          d.ID,
		Uid:         d.Uid,
		Title:       d.Title,
		Description: d.Description,
		BizId:       d.BizId,
		Biz:         d.Biz,
	}
}

func (c *caseSetRepo) toDomainCase(caseDao dao.Case) domain.Case {
	return domain.Case{
		Id:           caseDao.Id,
		Uid:          caseDao.Uid,
		Introduction: caseDao.Introduction,
		Labels:       caseDao.Labels.Val,
		Title:        caseDao.Title,
		Content:      caseDao.Content,
		GiteeRepo:    caseDao.GiteeRepo,
		GithubRepo:   caseDao.GithubRepo,
		Keywords:     caseDao.Keywords,
		Shorthand:    caseDao.Shorthand,
		Highlight:    caseDao.Highlight,
		Guidance:     caseDao.Guidance,
		Status:       domain.CaseStatus(caseDao.Status),
		Biz:          caseDao.Biz,
		BizId:        caseDao.BizId,
		Utime:        time.UnixMilli(caseDao.Utime),
		Ctime:        time.UnixMilli(caseDao.Ctime),
	}
}

func (c *caseSetRepo) toDomainCaseSet(caseSetDao dao.CaseSet) domain.CaseSet {
	return domain.CaseSet{
		ID:          caseSetDao.Id,
		Uid:         caseSetDao.Uid,
		Title:       caseSetDao.Title,
		Description: caseSetDao.Description,
		BizId:       caseSetDao.BizId,
		Biz:         caseSetDao.Biz,
		Utime:       caseSetDao.Utime,
	}
}
