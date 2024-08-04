package repository

import (
	"context"
	"golang.org/x/sync/errgroup"
	"time"

	"github.com/ecodeclub/ekit/slice"

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
)

type CaseRepo interface {
	// c端接口
	PubList(ctx context.Context, offset int, limit int) ([]domain.Case, error)
	GetPubByID(ctx context.Context, caseId int64) (domain.Case, error)
	GetPubByIDs(ctx context.Context, ids []int64) ([]domain.Case, error)
	// Sync 保存到制作库，而后同步到线上库
	Sync(ctx context.Context, ca domain.Case) (int64, error)
	// 管理端接口
	List(ctx context.Context, offset int, limit int) ([]domain.Case, error)
	Total(ctx context.Context) (int64, error)
	Save(ctx context.Context, ca domain.Case) (int64, error)
	GetById(ctx context.Context, caseId int64) (domain.Case, error)


	// ExcludeQuestions 分页接口，不含这些 id 的问题
	ExcludeCases(ctx context.Context, ids []int64, offset int, limit int) ([]domain.Case, int64, error)
}

type caseRepo struct {
	caseDao dao.CaseDAO
}

func (c *caseRepo) ExcludeCases(ctx context.Context, ids []int64, offset int, limit int) ([]domain.Case, int64, error) {
	var (
		eg   errgroup.Group
		cnt  int64
		data []dao.Case
	)
	eg.Go(func() error {
		var err error
		cnt, err = c.caseDao.NotInTotal(ctx, ids)
		return err
	})

	eg.Go(func() error {
		var err error
		data, err = c.caseDao.NotIn(ctx, ids, offset, limit)
		return err
	})
	err := eg.Wait()
	return slice.Map(data, func(idx int, src dao.Case) domain.Case{
		return c.toDomain(src)
	}), cnt, err
}

func (c *caseRepo) PubList(ctx context.Context, offset int, limit int) ([]domain.Case, error) {
	caseList, err := c.caseDao.PublishCaseList(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	domainCases := make([]domain.Case, 0, len(caseList))
	for _, ca := range caseList {
		domainCases = append(domainCases, c.toDomain(dao.Case(ca)))
	}
	return domainCases, nil
}

func (c *caseRepo) GetPubByID(ctx context.Context, caseId int64) (domain.Case, error) {
	caseInfo, err := c.caseDao.GetPublishCase(ctx, caseId)
	if err != nil {
		return domain.Case{}, err
	}
	return c.toDomain(dao.Case(caseInfo)), nil
}

func (c *caseRepo) GetPubByIDs(ctx context.Context, ids []int64) ([]domain.Case, error) {
	caseInfo, err := c.caseDao.GetPubByIDs(ctx, ids)
	return slice.Map(caseInfo, func(idx int, src dao.PublishCase) domain.Case {
		return c.toDomain(dao.Case(src))
	}), err
}

func (c *caseRepo) Sync(ctx context.Context, ca domain.Case) (int64, error) {
	caseModel := c.toEntity(ca)
	return c.caseDao.Sync(ctx, caseModel)
}

func (c *caseRepo) List(ctx context.Context, offset int, limit int) ([]domain.Case, error) {
	caseList, err := c.caseDao.List(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	domainCases := make([]domain.Case, 0, len(caseList))
	for _, ca := range caseList {
		domainCases = append(domainCases, c.toDomain(ca))
	}
	return domainCases, nil
}

func (c *caseRepo) Total(ctx context.Context) (int64, error) {
	return c.caseDao.Count(ctx)
}

func (c *caseRepo) Save(ctx context.Context, ca domain.Case) (int64, error) {
	return c.caseDao.Save(ctx, c.toEntity(ca))
}

func (c *caseRepo) GetById(ctx context.Context, caseId int64) (domain.Case, error) {
	ca, err := c.caseDao.GetCaseByID(ctx, caseId)
	if err != nil {
		return domain.Case{}, err
	}
	return c.toDomain(ca), err
}

func (c *caseRepo) toEntity(caseDomain domain.Case) dao.Case {
	labels := sqlx.JsonColumn[[]string]{
		Valid: len(caseDomain.Labels) > 0,
		Val:   caseDomain.Labels,
	}
	return dao.Case{
		Id:           caseDomain.Id,
		Uid:          caseDomain.Uid,
		Labels:       labels,
		Introduction: caseDomain.Introduction,
		Title:        caseDomain.Title,
		Content:      caseDomain.Content,
		CodeRepo:     caseDomain.CodeRepo,
		Keywords:     caseDomain.Keywords,
		Shorthand:    caseDomain.Shorthand,
		Highlight:    caseDomain.Highlight,
		Guidance:     caseDomain.Guidance,
		Status:       caseDomain.Status.ToUint8(),
	}
}

func (c *caseRepo) toDomain(caseDao dao.Case) domain.Case {
	return domain.Case{
		Id:           caseDao.Id,
		Uid:          caseDao.Uid,
		Introduction: caseDao.Introduction,
		Labels:       caseDao.Labels.Val,
		Title:        caseDao.Title,
		Content:      caseDao.Content,
		CodeRepo:     caseDao.CodeRepo,
		Keywords:     caseDao.Keywords,
		Shorthand:    caseDao.Shorthand,
		Highlight:    caseDao.Highlight,
		Guidance:     caseDao.Guidance,
		Utime:        time.UnixMilli(caseDao.Utime),
		Ctime:        time.UnixMilli(caseDao.Ctime),
		Status:       domain.CaseStatus(caseDao.Status),
	}
}

func NewCaseRepo(caseDao dao.CaseDAO) CaseRepo {
	return &caseRepo{
		caseDao: caseDao,
		// 后续接入缓存
	}
}
