package web

import (
	"fmt"
	"net/http"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/feedback/internal/domain"
	"github.com/ecodeclub/webook/internal/feedback/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc    service.Service
	logger *elog.Component
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{
		svc:    svc,
		logger: elog.DefaultLogger,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	// 列表 根据交互来, 先是未处理，然后是通过，最后是拒绝
	server.POST("/feedback/list", ginx.S(h.Permission), ginx.B[ListReq](h.List))
	// 未处理的个数
	server.GET("/feedback/pending-count", ginx.S(h.Permission), ginx.W(h.PendingCount))
	server.POST("/feedback/info", ginx.S(h.Permission), ginx.B[FeedBackID](h.Info))
	server.POST("/feedback/update-status", ginx.S(h.Permission),
		ginx.B[UpdateStatusReq](h.UpdateStatus))
	server.POST("/feedback/create", ginx.BS[CreateReq](h.Create))
}

func (h *Handler) PendingCount(ctx *ginx.Context) (ginx.Result, error) {
	count, err := h.svc.PendingCount(ctx)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: count,
	}, nil
}

func (h *Handler) List(ctx *ginx.Context, req ListReq) (ginx.Result, error) {
	data, err := h.svc.List(ctx, domain.FeedBack{
		BizID: req.BizID,
		Biz:   req.Biz,
	}, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toFeedBackList(data),
	}, nil
}
func (h *Handler) Info(ctx *ginx.Context, req FeedBackID) (ginx.Result, error) {
	detail, err := h.svc.Info(ctx, req.FID)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newFeedBack(detail),
	}, err
}
func (h *Handler) UpdateStatus(ctx *ginx.Context, req UpdateStatusReq) (ginx.Result, error) {
	err := h.svc.UpdateStatus(ctx, domain.FeedBack{
		ID:     req.FID,
		Status: domain.FeedBackStatus(req.Status),
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, err
}

func (h *Handler) Create(ctx *ginx.Context, req CreateReq, sess session.Session) (ginx.Result, error) {
	feedBack := req.FeedBack.toDomain()
	feedBack.UID = sess.Claims().Uid
	feedBack.Status = 0
	err := h.svc.Create(ctx, feedBack)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, err
}

func (h *Handler) Permission(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	if sess.Claims().Get("creator").StringOrDefault("") != "true" {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return ginx.Result{}, fmt.Errorf("非法访问创作中心 uid: %d", sess.Claims().Uid)
	}
	return ginx.Result{}, ginx.ErrNoResponse
}

func (h *Handler) toFeedBackList(data []domain.FeedBack) FeedBackList {
	return FeedBackList{
		FeedBacks: slice.Map(data, func(idx int, feedBack domain.FeedBack) FeedBack {
			return newFeedBack(feedBack)
		}),
	}
}
