package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/review/internal/domain"
	"github.com/ecodeclub/webook/internal/review/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc service.ReviewSvc
}

func NewHandler(svc service.ReviewSvc) *Handler {
	return &Handler{
		svc: svc,
	}
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	server.POST("/review/pub/list", ginx.B[Page](h.PubList))
}
func (h *Handler) MemberRoutes(server *gin.Engine) {
	server.POST("/review/pub/detail", ginx.B[DetailReq](h.PubDetail))
}
func (h *Handler) PubList(ctx *ginx.Context, req Page) (ginx.Result, error) {
	// 调用 service 层获取数据
	reviews, err := h.svc.PubList(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	list := slice.Map(reviews, func(idx int, src domain.Review) Review {
		return newReview(src)
	})
	// 返回结果
	return ginx.Result{
		Data: ReviewListResp{List: list},
	}, nil
}

// PubDetail 获取已发布的面试评测记录详情
func (h *Handler) PubDetail(ctx *ginx.Context, req DetailReq) (ginx.Result, error) {

	// 调用 service 层获取数据
	review, err := h.svc.PubInfo(ctx, req.ID)
	if err != nil {
		return systemErrorResult, err
	}

	// 转换为展示层对象并返回
	return ginx.Result{
		Data: newReview(review),
	}, nil
}
