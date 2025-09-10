package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/company"
	"github.com/ecodeclub/webook/internal/review/internal/domain"
	"github.com/ecodeclub/webook/internal/review/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type AdminHandler struct {
	svc        service.ReviewSvc
	companySvc company.Service
	logger     *elog.Component
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	server.POST("/review/save", ginx.BS[ReviewSaveReq](h.Save))
	server.POST("/review/list", ginx.B[Page](h.List))
	server.POST("/review/detail", ginx.B[DetailReq](h.Detail))
	server.POST("/review/publish", ginx.BS[ReviewSaveReq](h.Publish))
}

func NewAdminHandler(svc service.ReviewSvc, companySvc company.Service) *AdminHandler {
	return &AdminHandler{
		svc:        svc,
		companySvc: companySvc,
		logger:     elog.DefaultLogger,
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

func (h *AdminHandler) getCompanies(ctx *ginx.Context, reviews []domain.Review) map[int64]company.Company {
	ids := slice.Map(reviews, func(idx int, src domain.Review) int64 {
		return src.Company.ID
	})
	var err1 error
	companies, err1 := h.companySvc.GetByIds(ctx, ids)
	// 这个数据查询不到也不需要担心
	if err1 != nil {
		h.logger.Error("查询公司失败",
			elog.Any("ids", ids),
			elog.FieldErr(err1))
	}
	return companies
}
func (h *AdminHandler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	count, reviewList, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	companies := h.getCompanies(ctx, reviewList)
	return ginx.Result{
		Data: ReviewListResp{
			Total: count,
			List: slice.Map(reviewList, func(idx int, src domain.Review) Review {
				return newReviewWithCompany(src, companies[src.Company.ID])
			}),
		},
	}, nil
}

func (h *AdminHandler) Detail(ctx *ginx.Context, req DetailReq) (ginx.Result, error) {
	re, err := h.svc.Info(ctx, req.ID)
	if err != nil {
		return systemErrorResult, err
	}
	com, err1 := h.companySvc.GetById(ctx, re.Company.ID)
	if err1 != nil {
		h.logger.Error("查询公司信息失败",
			elog.Any("id", re.ID),
			elog.FieldErr(err1))
	}
	return ginx.Result{
		Data: newReviewWithCompany(re, com),
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
