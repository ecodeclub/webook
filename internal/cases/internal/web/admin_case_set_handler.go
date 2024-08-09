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

func (a *AdminCaseSetHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/case-sets")
	g.POST("/save", ginx.BS[CaseSet](a.SaveCaseSet))
	g.POST("/cases/save", ginx.B[UpdateCases](a.UpdateCases))
	g.POST("/list", ginx.B[Page](a.ListCaseSets))
	g.POST("/detail", ginx.B[CaseSetID](a.RetrieveCaseSetDetail))
	g.POST("/candidate", ginx.B[CandidateReq](a.Candidate))

}

func (a *AdminCaseSetHandler) Candidate(ctx *ginx.Context, req CandidateReq) (ginx.Result, error) {
	data, cnt, err := a.svc.GetCandidates(ctx, req.CSID, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	castList := toCaseList(data, cnt)
	return ginx.Result{
		Data: castList,
	}, nil
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
	err := a.svc.UpdateCases(ctx.Request.Context(), domain.CaseSet{
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

func toCaseList(data []domain.Case, cnt int64) CasesList {
	return CasesList{
		Total: cnt,
		Cases: slice.Map(data, func(idx int, src domain.Case) Case {
			return Case{
				Id:           src.Id,
				Title:        src.Title,
				Content:      src.Content,
				Labels:       src.Labels,
				GiteeRepo:    src.GiteeRepo,
				GithubRepo:   src.GithubRepo,
				Keywords:     src.Keywords,
				Shorthand:    src.Shorthand,
				Introduction: src.Introduction,
				Highlight:    src.Highlight,
				Guidance:     src.Guidance,
				Biz:          src.Biz,
				BizId:        src.BizId,
				Status:       src.Status.ToUint8(),
				Utime:        src.Utime.UnixMilli(),
			}
		}),
	}
}
