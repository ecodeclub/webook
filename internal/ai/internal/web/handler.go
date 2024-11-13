package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm"
	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid/v4"
)

type Handler struct {
	llmSvc llm.Service
}

func NewHandler(llmSvc llm.Service) *Handler {
	return &Handler{
		llmSvc: llmSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/llm/ask", ginx.BS(h.LLMAsk))
}

func (h *Handler) LLMAsk(ctx *ginx.Context, req LLMRequest, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	tid := shortuuid.New()
	aiReq := domain.LLMRequest{
		Uid:   uid,
		Tid:   tid,
		Biz:   req.Biz,
		Input: req.Input,
	}
	resp, err := h.llmSvc.Invoke(ctx, aiReq)
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
