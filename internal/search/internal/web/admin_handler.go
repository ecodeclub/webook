package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/search/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type AdminHandler struct {
	svc    service.SearchService
	logger *elog.Component
}

func NewAdminHandler(svc service.SearchService) *AdminHandler {
	return &AdminHandler{
		svc:    svc,
		logger: elog.DefaultLogger,
	}
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	server.POST("/search/list", ginx.B[SearchReq](h.List))
}

func (h *AdminHandler) List(ctx *ginx.Context, req SearchReq) (ginx.Result, error) {
	data, err := h.svc.Search(ctx, req.Offset, req.Limit, req.Keywords)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: NewSearchResult(data, nil),
	}, nil
}
