package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/review/internal/domain"
	"github.com/ecodeclub/webook/internal/review/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc     service.ReviewSvc
	intrSvc interactive.Service
	logger  *elog.Component
	sp      session.Provider
}

func NewHandler(svc service.ReviewSvc, intrSvc interactive.Service, sp session.Provider) *Handler {
	return &Handler{
		svc:     svc,
		intrSvc: intrSvc,
		logger:  elog.DefaultLogger,
		sp:      sp,
	}
}
func (h *Handler) getUid(gctx *ginx.Context) int64 {
	sess, err := h.sp.Get(gctx)
	if err != nil {
		// 没登录
		return 0
	}
	return sess.Claims().Uid
}
func (h *Handler) PublicRoutes(server *gin.Engine) {
	server.POST("/review/list", ginx.B[Page](h.PubList))
	server.POST("/review/detail", ginx.B[DetailReq](h.PubDetail))
}

func (h *Handler) PubList(ctx *ginx.Context, req Page) (ginx.Result, error) {
	// 调用 service 层获取数据
	uid := h.getUid(ctx)
	reviews, err := h.svc.PubList(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	intrs := map[int64]interactive.Interactive{}
	if len(reviews) > 0 {
		ids := slice.Map(reviews, func(idx int, src domain.Review) int64 {
			return src.ID
		})
		var err1 error
		intrs, err1 = h.intrSvc.GetByIds(ctx, "review", uid, ids)
		// 这个数据查询不到也不需要担心
		if err1 != nil {
			h.logger.Error("查询数据的点赞数据失败",
				elog.Any("ids", ids),
				elog.FieldErr(err))
		}
	}
	list := slice.Map(reviews, func(idx int, src domain.Review) Review {
		return newReviewWithInteractive(src, intrs[src.ID])
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

	var intr interactive.Interactive
	sess, err := h.sp.Get(ctx)
	if err == nil {
		uid := sess.Claims().Uid
		var err1 error
		intr, err1 = h.intrSvc.Get(ctx, "review", req.ID, uid)
		// 这个数据查询不到也不需要担心
		if err1 != nil {
			h.logger.Error("查询数据的点赞数据失败",
				elog.Any("id", req.ID),
				elog.FieldErr(err))
		}
	}

	// 转换为展示层对象并返回
	return ginx.Result{
		Data: newReviewWithInteractive(review, intr),
	}, nil
}
