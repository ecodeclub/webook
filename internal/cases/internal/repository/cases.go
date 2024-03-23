package repository

import (
	"context"
	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/cache"
	"github.com/ecodeclub/webook/internal/cases/internal/repository/dao"
	"github.com/gotomicro/ego/core/elog"
	"time"
)

type CaseRepo interface {
	// c端接口
	PubList(ctx context.Context, offset int, limit int) ([]domain.Case, error)
	PubTotal(ctx context.Context) (int64, error)
	GetPubByID(ctx context.Context, caseId int64) (domain.Case, error)
	// Sync 保存到制作库，而后同步到线上库
	Sync(ctx context.Context, ca *domain.Case) (int64, error)
	// 管理端接口
	List(ctx context.Context, offset int, limit int) ([]domain.Case, error)
	Total(ctx context.Context) (int64, error)
	Update(ctx context.Context, ca *domain.Case) error
	Create(ctx context.Context, ca *domain.Case) (int64, error)
	GetById(ctx context.Context, caseId int64) (domain.Case, error)
}

type caseRepo struct {
	caseDao   dao.CaseDAO
	caseCache cache.CaseCache
	logger    *elog.Component
}

func NewCaseRepo(caseDao dao.CaseDAO, caseCache cache.CaseCache) CaseRepo {
	return &caseRepo{
		caseDao:   caseDao,
		caseCache: caseCache,
	}
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

func (c *caseRepo) PubTotal(ctx context.Context) (int64, error) {
	res, err := c.caseCache.GetTotal(ctx)
	if err == nil {
		return res, err
	}
	res, err = c.caseDao.PublishCaseCount(ctx)
	if err != nil {
		return 0, err
	}
	err = c.caseCache.SetTotal(ctx, res)
	if err != nil {
		c.logger.Error("更新缓存中的总数失败", elog.FieldErr(err))
	}
	return res, nil
}

func (c *caseRepo) GetPubByID(ctx context.Context, caseId int64) (domain.Case, error) {
	caseInfo, err := c.caseDao.GetPublishCase(ctx, caseId)
	if err != nil {
		return domain.Case{}, err
	}
	return c.toDomain(dao.Case(caseInfo)), nil
}

func (c *caseRepo) Sync(ctx context.Context, ca *domain.Case) (int64, error) {
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

func (c *caseRepo) Update(ctx context.Context, ca *domain.Case) error {
	return c.caseDao.Update(ctx, c.toEntity(ca))
}

func (c *caseRepo) Create(ctx context.Context, ca *domain.Case) (int64, error) {
	return c.caseDao.Create(ctx, c.toEntity(ca))
}

func (c *caseRepo) GetById(ctx context.Context, caseId int64) (domain.Case, error) {
	ca, err := c.caseDao.GetCaseByID(ctx, caseId)
	if err != nil {
		return domain.Case{}, err
	}
	return c.toDomain(ca), err
}

func (c *caseRepo) toEntity(caseDomain *domain.Case) dao.Case {
	labels := sqlx.JsonColumn[[]string]{
		Valid: len(caseDomain.Labels) > 0,
		Val:   caseDomain.Labels,
	}
	return dao.Case{
		Id:        caseDomain.Id,
		Uid:       caseDomain.Uid,
		Labels:    labels,
		Title:     caseDomain.Title,
		Content:   caseDomain.Content,
		CodeRepo:  caseDomain.CodeRepo,
		Keywords:  caseDomain.Summary.Keywords,
		Shorthand: caseDomain.Summary.Shorthand,
		Highlight: caseDomain.Summary.Highlight,
		Guidance:  caseDomain.Summary.Guidance,
	}
}

func (c *caseRepo) toDomain(caseDao dao.Case) domain.Case {
	return domain.Case{
		Id:       caseDao.Id,
		Uid:      caseDao.Uid,
		Labels:   caseDao.Labels.Val,
		Title:    caseDao.Title,
		Content:  caseDao.Content,
		CodeRepo: caseDao.CodeRepo,
		Summary: domain.Summary{
			Keywords:  caseDao.Keywords,
			Shorthand: caseDao.Shorthand,
			Highlight: caseDao.Highlight,
			Guidance:  caseDao.Guidance,
		},
		Utime: time.UnixMilli(caseDao.Utime),
	}
}
