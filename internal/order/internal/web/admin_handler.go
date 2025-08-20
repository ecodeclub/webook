package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/order/internal/domain"
	"github.com/ecodeclub/webook/internal/order/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	svc service.Service
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/order")
	g.POST("/list", ginx.B[ListOrdersReq](h.List))
}
func NewAdminHandler(svc service.Service) *AdminHandler {
	return &AdminHandler{
		svc: svc,
	}
}

func (h *AdminHandler) List(ctx *ginx.Context, req ListOrdersReq) (ginx.Result, error) {
	count, list, err := h.svc.FindOrders(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: ListOrdersResp{
			Total: count,
			Orders: slice.Map(list, func(idx int, src domain.Order) Order {
				return toOrderVO(src)
			}),
		},
	}, nil
}
