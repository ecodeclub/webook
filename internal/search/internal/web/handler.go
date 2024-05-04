package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/search/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc    service.SearchService
	logger *elog.Component
}

func NewHandler(svc service.SearchService) *Handler {
	return &Handler{
		svc:    svc,
		logger: elog.DefaultLogger,
	}
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	server.POST("/search/list", ginx.B[SearchReq](h.List))
}

func (h *Handler) List(ctx *ginx.Context, req SearchReq) (ginx.Result, error) {
	data, err := h.svc.Search(ctx, req.KeyWords)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: NewSearchResult(data),
	}, nil
}
