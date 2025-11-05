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
	// 使用标准库上下文以保留超时/取消控制，避免并发使用 *gin.Context
	stdCtx := ctx.Request.Context()
	data, err := h.svc.Search(stdCtx, req.Offset, req.Limit, req.Keywords)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: NewSearchResult(data, nil),
	}, nil
}
