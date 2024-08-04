package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminCaseSetHandler struct {
	svc service.CaseSetService
}

func NewAdminCaseSetHandler(svc service.CaseSetService) *AdminCaseSetHandler {
	return &AdminCaseSetHandler{svc: svc}
}

func (a *AdminCaseSetHandler) PublicRoutes(server *gin.Engine) {
}

func (a *AdminCaseSetHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/case-sets")
	g.POST("/save", ginx.BS[CaseSet](a.SaveCaseSet))
	g.POST("/cases/save", ginx.B[UpdateCases](a.UpdateCases))
	g.POST("/list", ginx.B[Page](a.ListCaseSets))
	g.POST("/detail", ginx.B[CaseSetID](a.RetrieveCaseSetDetail))
}

func (a *AdminCaseSetHandler) SaveCaseSet(ctx *ginx.Context,
	req CaseSet,
	sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	id, err := a.svc.Save(ctx, domain.CaseSet{
		ID:          req.Id,
		Uid:         uid,
		Title:       req.Title,
		Description: req.Description,
		Biz:         req.Biz,
		BizId:       req.BizId,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (a *AdminCaseSetHandler) UpdateCases(ctx *ginx.Context, req UpdateCases) (ginx.Result, error) {
	cs := slice.Map(req.CIDs, func(idx int, src int64) domain.Case {
		return domain.Case{
			Id: src,
		}
	})
	err := a.svc.UpdateCases(ctx, domain.CaseSet{
		ID:    req.CSID,
		Cases: cs,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil

}

func (a *AdminCaseSetHandler) ListCaseSets(ctx *ginx.Context, req Page) (ginx.Result, error) {
	list, count, err := a.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: CaseSetList{
			Total: count,
			CaseSets: slice.Map(list, func(idx int, src domain.CaseSet) CaseSet {
				return newCaseSet(src)
			}),
		},
	}, nil
}

func (a *AdminCaseSetHandler) RetrieveCaseSetDetail(ctx *ginx.Context, req CaseSetID) (ginx.Result, error) {
	detail, err := a.svc.Detail(ctx, req.ID)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newCaseSet(detail),
	}, nil

}
