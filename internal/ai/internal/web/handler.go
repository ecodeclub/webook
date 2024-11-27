package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	generalSvc service.GeneralService
	jdSvc      service.JDService
}

func NewHandler(generalSvc service.GeneralService, jdSvc service.JDService) *Handler {
	return &Handler{
		generalSvc: generalSvc,
		jdSvc:      jdSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/ai/ask", ginx.BS(h.LLMAsk))
	server.POST("/ai/analysis_jd", ginx.BS(h.AnalysisJd))
}

func (h *Handler) LLMAsk(ctx *ginx.Context, req LLMRequest, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	resp, err := h.generalSvc.LLMAsk(ctx, uid, req.Biz, req.Input)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: LLMResponse{
			Amount:    resp.Amount,
			RawResult: resp.Answer,
		},
	}, nil
}

func (h *Handler) AnalysisJd(ctx *ginx.Context, req JDRequest, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	resp, err := h.jdSvc.Evaluate(ctx, uid, req.JD)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: JDResponse{
			Amount:    resp.Amount,
			TechScore: h.newJD(resp.TechScore),
			BizScore:  h.newJD(resp.BizScore),
			PosScore:  h.newJD(resp.PosScore),
		},
	}, nil

}

func (h *Handler) newJD(jd *domain.JDEvaluation) *JDEvaluation {
	return &JDEvaluation{
		Score:    jd.Score,
		Analysis: jd.Analysis,
	}
}
