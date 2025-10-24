package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/label/internal/domain"
	"github.com/ecodeclub/webook/internal/label/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	g := server.Group("/label")
	g.GET("/system", ginx.W(h.SystemLabels))
}

func (h *Handler) SystemLabels(ctx *ginx.Context) (ginx.Result, error) {
	labels, err := h.svc.SystemLabels(ctx)
	if err != nil {
		return ginx.Result{}, err
	}
	return ginx.Result{
		Data: slice.Map(labels, func(idx int, src domain.Label) Label {
			return Label{
				Id:   src.Id,
				Name: src.Name,
			}
		}),
	}, nil
}
