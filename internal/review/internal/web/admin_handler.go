package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/review/internal/domain"
	"github.com/ecodeclub/webook/internal/review/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	svc service.ReviewSvc
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	server.POST("/review/save", ginx.BS[ReviewSaveReq](h.Save))
	server.POST("/review/list", ginx.B[Page](h.List))
	server.POST("/review/detail", ginx.B[DetailReq](h.Detail))
	server.POST("/review/publish", ginx.BS[ReviewSaveReq](h.Publish))
}

func NewAdminHandler(svc service.ReviewSvc) *AdminHandler {
	return &AdminHandler{
		svc: svc,
	}
}

func (h *AdminHandler) Save(ctx *ginx.Context, req ReviewSaveReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	review := req.Review.toDomain()
	review.Uid = uid
	id, err := h.svc.Save(ctx, review)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	count, reviewList, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: ReviewListResp{
			Total: count,
			List: slice.Map(reviewList, func(idx int, src domain.Review) Review {
				return newReview(src)
			}),
		},
	}, nil
}

func (h *AdminHandler) Detail(ctx *ginx.Context, req DetailReq) (ginx.Result, error) {
	re, err := h.svc.Info(ctx, req.ID)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newReview(re),
	}, nil
}

func (h *AdminHandler) Publish(ctx *ginx.Context, req ReviewSaveReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	review := req.Review.toDomain()
	review.Uid = uid
	id, err := h.svc.Publish(ctx, review)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}
