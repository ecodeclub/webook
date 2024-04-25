package web

import (
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/search/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc    service.SearchSvc
	logger *elog.Component
}

func NewHandler(svc service.SearchSvc) *Handler {
	return &Handler{
		svc:    svc,
		logger: elog.DefaultLogger,
	}
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	server.POST("/search/list", ginx.B[SearchReq](h.List))
	server.POST("/search/list/biz", ginx.B[SearchBizReq](h.BizList))
}

func (h *Handler) List(ctx *ginx.Context, req SearchReq) (ginx.Result, error) {
	// 制作库不需要统计总数
	data, err := h.svc.Search(ctx, req.KeyWords)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: NewSearchResult(data),
	}, nil
}
func (h *Handler) BizList(ctx *ginx.Context, req SearchBizReq) (ginx.Result, error) {
	// 制作库不需要统计总数
	data, err := h.svc.SearchWithBiz(ctx, req.Biz, req.KeyWords)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: NewSearchResult(data),
	}, nil
}
