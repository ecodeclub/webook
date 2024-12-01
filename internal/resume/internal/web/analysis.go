package web

import (
	"errors"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/resume/internal/errs"
	"github.com/ecodeclub/webook/internal/resume/internal/service"
	"github.com/gin-gonic/gin"
)

type AnalysisHandler struct {
	svc service.AnalysisService
}

func NewAnalysisHandler(svc service.AnalysisService) *AnalysisHandler {
	return &AnalysisHandler{
		svc: svc,
	}
}

func (h *AnalysisHandler) MemberRoutes(server *gin.Engine) {
	g := server.Group("/resume/analysis")
	g.POST("", ginx.BS(h.Analysis))
}

func (h *AnalysisHandler) Analysis(ctx *ginx.Context, req AnalysisReq, sess session.Session) (ginx.Result, error) {
	analysis, err := h.svc.Analysis(ctx, sess.Claims().Uid, req.Resume)
	switch {
	case errors.Is(err, service.ErrInsufficientCredit):
		return ginx.Result{
			Code: errs.InsufficientCredit.Code,
			Msg:  errs.InsufficientCredit.Msg,
		}, nil

	case err == nil:
		return ginx.Result{
			Data: AnalysisResp{
				Amount:         analysis.Amount,
				RewriteProject: analysis.RewriteProject,
				RewriteSkills:  analysis.RewriteSkills,
				RewriteJobs:    analysis.RewriteJobs,
			},
		}, nil
	default:
		return systemErrorResult, err
	}
}
