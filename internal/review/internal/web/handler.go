// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/company"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/review/internal/domain"
	"github.com/ecodeclub/webook/internal/review/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/sync/errgroup"
)

type Handler struct {
	svc        service.ReviewSvc
	intrSvc    interactive.Service
	companySvc company.Service
	logger     *elog.Component
	sp         session.Provider
}

func NewHandler(svc service.ReviewSvc, intrSvc interactive.Service,
	companySvc company.Service,
	sp session.Provider) *Handler {
	return &Handler{
		svc:        svc,
		intrSvc:    intrSvc,
		logger:     elog.DefaultLogger,
		companySvc: companySvc,
		sp:         sp,
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
	companies := make(map[int64]company.Company, len(reviews))
	var eg errgroup.Group
	if len(reviews) > 0 {
		eg.Go(func() error {
			intrs = h.getIntrs(ctx, uid, reviews)
			return nil
		})
		eg.Go(func() error {
			companies = h.getCompanies(ctx, reviews)
			return nil
		})
	}
	if err = eg.Wait(); err != nil {
		return systemErrorResult, err
	}
	list := slice.Map(reviews, func(idx int, src domain.Review) Review {
		return newCompleteReview(src, intrs[src.ID], companies[src.Company.ID])
	})
	// 返回结果
	return ginx.Result{
		Data: ReviewListResp{List: list},
	}, nil
}
func (h *Handler) getIntrs(ctx *ginx.Context, uid int64, reviews []domain.Review) map[int64]interactive.Interactive {
	ids := slice.Map(reviews, func(idx int, src domain.Review) int64 {
		return src.ID
	})
	var err1 error
	intrs, err1 := h.intrSvc.GetByIds(ctx, "review", uid, ids)
	// 这个数据查询不到也不需要担心
	if err1 != nil {
		h.logger.Error("查询数据的点赞数据失败",
			elog.Any("ids", ids),
			elog.FieldErr(err1))
	}
	return intrs
}

func (h *Handler) getCompanies(ctx *ginx.Context, reviews []domain.Review) map[int64]company.Company {
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

// PubDetail 获取已发布的面试评测记录详情
func (h *Handler) PubDetail(ctx *ginx.Context, req DetailReq) (ginx.Result, error) {
	// 调用 service 层获取数据
	review, err := h.svc.PubInfo(ctx, req.ID)
	if err != nil {
		return systemErrorResult, err
	}
	// 获取company
	com, err1 := h.companySvc.GetById(ctx, review.Company.ID)
	if err1 != nil {
		h.logger.Error("查询公司信息失败",
			elog.Any("id", review.Company.ID),
			elog.FieldErr(err1))
	}
	var intr interactive.Interactive
	sess, err := h.sp.Get(ctx)
	if err == nil {
		uid := sess.Claims().Uid
		intr = h.getIntr(ctx, uid, review)
	}
	// 转换为展示层对象并返回
	return ginx.Result{
		Data: newCompleteReview(review, intr, com),
	}, nil
}

func (h *Handler) getIntr(ctx *ginx.Context, uid int64, review domain.Review) interactive.Interactive {
	intr, err1 := h.intrSvc.Get(ctx, "review", review.ID, uid)
	// 这个数据查询不到也不需要担心
	if err1 != nil {
		h.logger.Error("查询数据的点赞数据失败",
			elog.Any("id", review.ID),
			elog.FieldErr(err1))
	}
	return intr
}
