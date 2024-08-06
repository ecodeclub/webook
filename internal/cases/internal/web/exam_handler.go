package web

import (
	"errors"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases/internal/errs"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/gin-gonic/gin"
)

type ExamineHandler struct {
	svc service.ExamineService
}

func NewExamineHandler(svc service.ExamineService) *ExamineHandler {
	return &ExamineHandler{
		svc: svc,
	}
}

func (h *ExamineHandler) MemberRoutes(server *gin.Engine) {
	g := server.Group("/cases/examine")
	g.POST("", ginx.BS(h.Examine))
}

func (h *ExamineHandler) Examine(ctx *ginx.Context, req ExamineReq, sess session.Session) (ginx.Result, error) {
	res, err := h.svc.Examine(ctx, sess.Claims().Uid, req.Cid, req.Input)
	switch {
	case errors.Is(err, service.ErrInsufficientCredit):
		return ginx.Result{
			Code: errs.InsufficientCredits.Code,
			Msg:  errs.InsufficientCredits.Msg,
		}, nil

	case err == nil:
		return ginx.Result{
			Data: newExamineResult(res),
		}, nil
	default:
		return systemErrorResult, err
	}
}
