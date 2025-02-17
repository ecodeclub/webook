package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/cases/internal/repository/cache"
	"github.com/gotomicro/ego/core/elog"
	"github.com/pkg/errors"

	"golang.org/x/sync/errgroup"

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
	PubCount(ctx context.Context) (int64, error)
	// Sync 保存到制作库，而后同步到线上库
	Sync(ctx context.Context, ca domain.Case) (int64, error)
	// 管理端接口
	Ids(ctx context.Context) ([]int64, error)
	List(ctx context.Context, offset int, limit int) ([]domain.Case, error)
	Total(ctx context.Context) (int64, error)
	Save(ctx context.Context, ca domain.Case) (int64, error)
	GetById(ctx context.Context, caseId int64) (domain.Case, error)

	// Exclude 分页接口，不含这些 id 的问题
	Exclude(ctx context.Context, ids []int64, offset int, limit int) ([]domain.Case, int64, error)
}

type caseRepo struct {
	caseDao   dao.CaseDAO
	caseCache cache.CaseCache
	logger    *elog.Component
}

func (c *caseRepo) PubCount(ctx context.Context) (int64, error) {
	total, cacheErr := c.caseCache.GetTotal(ctx, domain.DefaultBiz)
	if cacheErr == nil {
		return total, nil
	}
	total, err := c.caseDao.PublishCaseCount(ctx, domain.DefaultBiz)
	if err != nil {
		return 0, err
	}
	cacheErr = c.caseCache.SetTotal(ctx, domain.DefaultBiz, total)
	if cacheErr != nil {
		// 记录一下日志
		c.logger.Error("记录缓存失败", elog.FieldErr(cacheErr))
	}
	return total, nil
}

func (c *caseRepo) Ids(ctx context.Context) ([]int64, error) {
	return c.caseDao.Ids(ctx)
}

func (c *caseRepo) Exclude(ctx context.Context, ids []int64, offset int, limit int) ([]domain.Case, int64, error) {
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
	return slice.Map(data, func(idx int, src dao.Case) domain.Case {
		return c.toDomain(src)
	}), cnt, err
}

func (c *caseRepo) PubList(ctx context.Context, offset int, limit int) ([]domain.Case, error) {
	// 检查是否在缓存范围内
	if c.checkTop50(offset, limit) {
		// 尝试从缓存获取
		cases, err := c.caseCache.GetCases(ctx, domain.DefaultBiz)
		if err == nil {
			return c.getCasesFromCache(cases, offset, limit), nil
		}
		domainCases, err := c.cacheList(ctx, domain.DefaultBiz)
		if err != nil {
			return domainCases, err
		}
		return c.getCasesFromCache(domainCases, offset, limit), nil
	}

	// 超出缓存范围，直接查询数据库
	caseList, err := c.caseDao.PublishCaseList(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	return slice.Map(caseList, func(idx int, src dao.PublishCase) domain.Case {
		return c.toDomain(dao.Case(src))
	}), nil
}

func (c *caseRepo) GetPubByID(ctx context.Context, caseId int64) (domain.Case, error) {
	ca, eerr := c.caseCache.GetCase(ctx, caseId)
	if eerr == nil {
		// 命中缓存
		return ca, nil
	}
	if !errors.Is(eerr, cache.ErrCaseNotFound) {
		// 记录一下日志
		c.logger.Error("案例获取缓存失败", elog.FieldErr(eerr), elog.Int64("cid", caseId))
	}

	daoCa, err := c.caseDao.GetPublishCase(ctx, caseId)
	if err != nil {
		return domain.Case{}, err
	}
	ca = c.toDomain(dao.Case(daoCa))
	eerr = c.caseCache.SetCase(ctx, ca)
	if eerr != nil {
		// 记录一下日志
		c.logger.Error("案例设置缓存失败", elog.FieldErr(eerr), elog.Int64("cid", caseId))
	}
	return ca, nil
}

func (c *caseRepo) GetPubByIDs(ctx context.Context, ids []int64) ([]domain.Case, error) {
	caseInfo, err := c.caseDao.GetPubByIDs(ctx, ids)
	return slice.Map(caseInfo, func(idx int, src dao.PublishCase) domain.Case {
		return c.toDomain(dao.Case(src))
	}), err
}

func (c *caseRepo) Sync(ctx context.Context, ca domain.Case) (int64, error) {
	caseModel := c.toEntity(ca)
	daoCa, err := c.caseDao.Sync(ctx, caseModel)
	if err != nil {
		return 0, err
	}

	// 获取最新数据并更新缓存
	domainCase := c.toDomain(daoCa)
	eerr := c.caseCache.SetCase(ctx, domainCase)
	if eerr != nil {
		c.logger.Error("案例设置缓存失败", elog.FieldErr(eerr), elog.Int64("cid", daoCa.Id))
	}

	// 更新前50条列表缓存
	_, cacheErr := c.cacheList(ctx, domainCase.Biz)
	if cacheErr != nil {
		c.logger.Error("更新案例列表缓存失败", elog.FieldErr(cacheErr), elog.String("biz", domainCase.Biz))
	}

	return daoCa.Id, nil
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
		GithubRepo:   caseDomain.GithubRepo,
		GiteeRepo:    caseDomain.GiteeRepo,
		Keywords:     caseDomain.Keywords,
		Shorthand:    caseDomain.Shorthand,
		Highlight:    caseDomain.Highlight,
		Guidance:     caseDomain.Guidance,
		Biz:          caseDomain.Biz,
		BizId:        caseDomain.BizId,
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
		GithubRepo:   caseDao.GithubRepo,
		GiteeRepo:    caseDao.GiteeRepo,
		Keywords:     caseDao.Keywords,
		Shorthand:    caseDao.Shorthand,
		Highlight:    caseDao.Highlight,
		Guidance:     caseDao.Guidance,
		Biz:          caseDao.Biz,
		BizId:        caseDao.BizId,
		Utime:        time.UnixMilli(caseDao.Utime),
		Ctime:        time.UnixMilli(caseDao.Ctime),
		Status:       domain.CaseStatus(caseDao.Status),
	}
}

func NewCaseRepo(caseDao dao.CaseDAO, caseCache cache.CaseCache) CaseRepo {
	return &caseRepo{
		caseDao: caseDao,
		// 后续接入缓存
		caseCache: caseCache,
		logger:    elog.DefaultLogger,
	}
}

// 新增缓存范围检查方法
const (
	cacheMax = 50
	cacheMin = 0
)

func (c *caseRepo) checkTop50(offset, limit int) bool {
	last := offset + limit
	return last <= cacheMax
}

// 新增从缓存数据分页方法
func (c *caseRepo) getCasesFromCache(cases []domain.Case, offset, limit int) []domain.Case {
	if offset >= len(cases) {
		return []domain.Case{}
	}
	remain := len(cases) - offset
	if remain > limit {
		remain = limit
	}
	res := make([]domain.Case, 0, remain)
	for i := offset; i < offset+remain; i++ {
		res = append(res, cases[i])
	}
	return res
}

func (c *caseRepo) cacheList(ctx context.Context, biz string) ([]domain.Case, error) {
	caseList, err := c.caseDao.PublishCaseList(ctx, cacheMin, cacheMax)
	if err != nil {
		return nil, err
	}
	domainCases := slice.Map(caseList, func(idx int, src dao.PublishCase) domain.Case {
		return c.toDomain(dao.Case(src))
	})
	cacheErr := c.caseCache.SetCases(ctx, biz, domainCases)
	if cacheErr != nil {
		c.logger.Error("案例列表设置缓存失败", elog.FieldErr(cacheErr), elog.String("biz", biz))
	}
	return domainCases, nil
}
