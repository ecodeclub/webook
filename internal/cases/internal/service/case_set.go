package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/cases/internal/event"
	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/repository"
	"golang.org/x/sync/errgroup"
)

//go:generate mockgen -source=./case_set.go -destination=../../mocks/case_set.mock.go -package=casemocks -typed CaseSetService
type CaseSetService interface {
	Save(ctx context.Context, set domain.CaseSet) (int64, error)
	UpdateCases(ctx context.Context, set domain.CaseSet) error
	List(ctx context.Context, offset, limit int) ([]domain.CaseSet, int64, error)
	Detail(ctx context.Context, id int64) (domain.CaseSet, error)
	GetByIds(ctx context.Context, ids []int64) ([]domain.CaseSet, error)
	// GetByIdsWithCases 会查询关联的 Case，但是目前只是发返回了 ID
	GetByIdsWithCases(ctx context.Context, ids []int64) ([]domain.CaseSet, error)

	ListByBiz(ctx context.Context, offset, limit int, biz string) ([]domain.CaseSet, error)
	ListDefault(ctx context.Context, offset, limit int) (int64, []domain.CaseSet, error)
	GetByBiz(ctx context.Context, biz string, bizId int64) (domain.CaseSet, error)
	GetCandidates(ctx context.Context, id int64, offset int, limit int) ([]domain.Case, int64, error)
}

type caseSetSvc struct {
	repo     repository.CaseSetRepository
	caRepo   repository.CaseRepo
	producer event.InteractiveEventProducer
	logger   *elog.Component
}

func NewCaseSetService(repo repository.CaseSetRepository,
	caRepo repository.CaseRepo,
	producer event.InteractiveEventProducer,
) CaseSetService {
	return &caseSetSvc{
		repo:     repo,
		caRepo:   caRepo,
		producer: producer,
		logger:   elog.DefaultLogger,
	}
}

func (c *caseSetSvc) GetCandidates(ctx context.Context, id int64, offset int, limit int) ([]domain.Case, int64, error) {
	cs, err := c.repo.GetByID(ctx, id)
	if err != nil {
		return nil, 0, err
	}
	cids := slice.Map(cs.Cases, func(idx int, src domain.Case) int64 {
		return src.Id
	})
	// 在 NOT IN 查询里面，如果要是 cids 没有元素，那么会变成 NOT IN （NULL）
	// 结果就是一个都查询不到，所以这是一个 tricky 的写法
	// 不然走 if-else 代码就很难看
	if len(cids) == 0 {
		cids = append(cids, -1)
	}
	return c.caRepo.Exclude(ctx, cids, offset, limit)
}

func (c *caseSetSvc) ListDefault(ctx context.Context, offset, limit int) (int64, []domain.CaseSet, error) {
	var (
		eg    errgroup.Group
		total int64
		css   []domain.CaseSet
	)
	eg.Go(func() error {
		var err error
		css, err = c.repo.ListByBiz(ctx, offset, limit, domain.DefaultBiz)
		return err
	})
	eg.Go(func() error {
		var err error
		total, err = c.repo.CountByBiz(ctx, domain.DefaultBiz)
		return err
	})
	return total, css, eg.Wait()
}

func (c *caseSetSvc) ListByBiz(ctx context.Context, offset, limit int, biz string) ([]domain.CaseSet, error) {
	return c.repo.ListByBiz(ctx, offset, limit, biz)
}

func (c *caseSetSvc) GetByBiz(ctx context.Context, biz string, bizId int64) (domain.CaseSet, error) {
	return c.repo.GetByBiz(ctx, biz, bizId)
}

func (c *caseSetSvc) Save(ctx context.Context, set domain.CaseSet) (int64, error) {
	var id = set.ID
	var err error
	if set.ID > 0 {
		err = c.repo.UpdateNonZero(ctx, set)
	} else {
		id, err = c.repo.CreateCaseSet(ctx, set)
	}
	return id, err
}

func (c *caseSetSvc) UpdateCases(ctx context.Context, set domain.CaseSet) error {
	return c.repo.UpdateCases(ctx, set)
}

func (c *caseSetSvc) List(ctx context.Context, offset, limit int) ([]domain.CaseSet, int64, error) {
	var eg errgroup.Group
	var sets []domain.CaseSet
	var total int64
	eg.Go(func() error {
		var eerr error
		sets, eerr = c.repo.List(ctx, offset, limit)
		return eerr
	})
	eg.Go(func() error {
		var eerr error
		total, eerr = c.repo.Total(ctx)
		return eerr
	})

	if err := eg.Wait(); err != nil {
		return nil, 0, err
	}
	return sets, total, nil
}

func (c *caseSetSvc) Detail(ctx context.Context, id int64) (domain.CaseSet, error) {
	res, err := c.repo.GetByID(ctx, id)
	if err == nil {
		// 同步异步的区别不大
		err1 := c.producer.Produce(ctx, event.NewViewCntEvent(id, domain.BizCaseSet))
		if err1 != nil {
			c.logger.Error("更新观看计数失败",
				elog.Int64("id", id), elog.FieldErr(err))
		}
	}
	return res, err
}

func (c *caseSetSvc) GetByIds(ctx context.Context, ids []int64) ([]domain.CaseSet, error) {
	return c.repo.GetByIDs(ctx, ids)
}

func (c *caseSetSvc) GetByIdsWithCases(ctx context.Context, ids []int64) ([]domain.CaseSet, error) {
	return c.repo.GetByIDsWithCases(ctx, ids)
}
