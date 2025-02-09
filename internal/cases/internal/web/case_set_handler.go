package web

import (
	"context"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/sync/errgroup"
)

type CaseSetHandler struct {
	svc        service.CaseSetService
	examineSvc service.ExamineService
	logger     *elog.Component
	intrSvc    interactive.Service
	sp         session.Provider
}

func NewCaseSetHandler(
	svc service.CaseSetService,
	examineSvc service.ExamineService,
	intrSvc interactive.Service,
	sp session.Provider,
) *CaseSetHandler {
	return &CaseSetHandler{
		svc:        svc,
		intrSvc:    intrSvc,
		examineSvc: examineSvc,
		logger:     elog.DefaultLogger,
		sp:         sp,
	}
}

func (h *CaseSetHandler) PublicRoutes(server *gin.Engine) {
	g := server.Group("/case-sets")
	g.POST("/list", ginx.B[Page](h.ListCaseSets))
}

func (h *CaseSetHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/case-sets")
	g.POST("/detail", ginx.BS(h.RetrieveCaseSetDetail))
	g.POST("/detail/biz", ginx.BS(h.GetDetailByBiz))
}
func (h *CaseSetHandler) getUid(gctx *ginx.Context) int64 {
	sess, err := h.sp.Get(gctx)
	if err != nil {
		// 没登录
		return 0
	}
	return sess.Claims().Uid
}

// ListCaseSets 展示个人案例集
func (h *CaseSetHandler) ListCaseSets(ctx *ginx.Context, req Page) (ginx.Result, error) {
	uid := h.getUid(ctx)
	count, data, err := h.svc.ListDefault(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	// 查询点赞收藏记录
	intrs := map[int64]interactive.Interactive{}
	if len(data) > 0 {
		ids := slice.Map(data, func(idx int, src domain.CaseSet) int64 {
			return src.ID
		})
		var err1 error
		intrs, err1 = h.intrSvc.GetByIds(ctx, "caseSet", uid, ids)
		// 这个数据查询不到也不需要担心
		if err1 != nil {
			h.logger.Error("查询案例集的点赞数据失败",
				elog.Any("ids", ids),
				elog.FieldErr(err))
		}
	}
	return ginx.Result{
		Data: CaseSetList{
			Total: count,
			CaseSets: slice.Map(data, func(idx int, src domain.CaseSet) CaseSet {
				qs := newCaseSet(src)
				qs.Interactive = newInteractive(intrs[src.ID])
				return qs
			}),
		},
	}, nil
}

// RetrieveCaseSetDetail 案例集详情
func (h *CaseSetHandler) RetrieveCaseSetDetail(
	ctx *ginx.Context,
	req CaseSetID, sess session.Session) (ginx.Result, error) {

	data, err := h.svc.Detail(ctx.Request.Context(), req.ID)
	if err != nil {
		return systemErrorResult, err
	}
	return h.getDetail(ctx, sess.Claims().Uid, data)
}

func (h *CaseSetHandler) GetDetailByBiz(
	ctx *ginx.Context,
	req BizReq, sess session.Session) (ginx.Result, error) {
	data, err := h.svc.GetByBiz(ctx, req.Biz, req.BizId)
	if err != nil {
		return systemErrorResult, err
	}
	return h.getDetail(ctx, sess.Claims().Uid, data)
}

func (h *CaseSetHandler) getDetail(
	ctx context.Context,
	uid int64,
	cs domain.CaseSet) (ginx.Result, error) {
	var (
		eg          errgroup.Group
		intr        interactive.Interactive
		caseIntrMap map[int64]interactive.Interactive
		resultMap   map[int64]domain.ExamineCaseResult
	)

	eg.Go(func() error {
		var err error
		intr, err = h.intrSvc.Get(ctx, domain.BizCaseSet, cs.ID, uid)
		return err
	})
	eg.Go(func() error {
		var err error
		cids := cs.Cids()
		caseIntrMap, err = h.intrSvc.GetByIds(ctx, domain.BizCase, uid, cids)
		return err
	})

	eg.Go(func() error {
		var err error
		resultMap, err = h.examineSvc.GetResults(ctx, uid, cs.Cids())
		return err
	})

	err := eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: h.toCaseSetVO(cs, intr, caseIntrMap, resultMap),
	}, nil
}

func (h *CaseSetHandler) toCaseSetVO(
	set domain.CaseSet,
	intr interactive.Interactive,
	caseIntrMap map[int64]interactive.Interactive,
	results map[int64]domain.ExamineCaseResult) CaseSet {
	cs := newCaseSet(set)
	cs.Cases = h.toCaseVO(set.Cases, results, caseIntrMap)
	cs.Interactive = newInteractive(intr)
	return cs
}

func (h *CaseSetHandler) toCaseVO(cases []domain.Case,
	results map[int64]domain.ExamineCaseResult,
	caseIntrMap map[int64]interactive.Interactive) []Case {
	return slice.Map(cases, func(idx int, src domain.Case) Case {
		ca := newCase(src)
		res := results[ca.Id]
		ca.Interactive = newInteractive(caseIntrMap[ca.Id])
		ca.ExamineResult = res.Result.ToUint8()
		return ca
	})
}
