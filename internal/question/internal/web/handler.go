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
	"fmt"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/sync/errgroup"
)

type Handler struct {
	logger     *elog.Component
	intrSvc    interactive.Service
	examineSvc service.ExamineService
	svc        service.Service
	permSvc    permission.Service
}

func NewHandler(intrSvc interactive.Service,
	examineSvc service.ExamineService,
	permSvc permission.Service,
	svc service.Service) *Handler {
	return &Handler{intrSvc: intrSvc,
		permSvc:    permSvc,
		examineSvc: examineSvc, svc: svc}
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	// 下次发版要删除这个 pub
	server.POST("/question/pub/list", ginx.B[Page](h.PubList))
	server.POST("/question/list", ginx.B[Page](h.PubList))
}

func (h *Handler) MemberRoutes(server *gin.Engine) {
	// 下次发版要删除这个 pub
	server.POST("/question/pub/detail", ginx.BS[Qid](h.PubDetail))
	server.POST("/question/detail", ginx.BS[Qid](h.PubDetail))
}

func (h *Handler) PubDetail(ctx *ginx.Context,
	req Qid, sess session.Session) (ginx.Result, error) {
	var (
		eg      errgroup.Group
		detail  domain.Question
		intr    interactive.Interactive
		examine domain.Result
	)
	uid := sess.Claims().Uid
	eg.Go(func() error {
		var err error
		detail, err = h.svc.PubDetail(ctx, req.Qid)
		if err != nil {
			return fmt.Errorf("查找面试题详情失败 %w", err)
		}
		// 非八股文，我们需要判定是否有权限
		// 暂时在这里聚合
		if !detail.IsBaguwen() {
			var ok bool
			ok, err = h.permSvc.HasPermission(ctx, permission.Permission{
				Uid:   uid,
				Biz:   detail.Biz,
				BizID: detail.BizId,
			})
			if err != nil {
				return fmt.Errorf("判定用户是否有权限失败 %w", err)
			}
			if !ok {
				return fmt.Errorf("用户不具有面试题对应业务的权限 uid %d, biz: %s, bizId: %d", uid, detail.Biz, detail.BizId)
			}
		}
		return nil
	})

	eg.Go(func() error {
		var err error
		intr, err = h.intrSvc.Get(ctx, domain.QuestionBiz, req.Qid, uid)
		return err
	})

	eg.Go(func() error {
		var err error
		examine, err = h.examineSvc.QuestionResult(ctx, uid, req.Qid)
		return err
	})
	err := eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}

	que := newQuestion(detail, intr)
	que.ExamineResult = examine.ToUint8()
	return ginx.Result{
		Data: que,
	}, err
}

func (h *Handler) PubList(ctx *ginx.Context, req Page) (ginx.Result, error) {
	data, err := h.svc.PubList(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	// 查询点赞收藏记录
	intrs := map[int64]interactive.Interactive{}
	if len(data) > 0 {
		ids := slice.Map(data, func(idx int, src domain.Question) int64 {
			return src.Id
		})
		var err1 error
		intrs, err1 = h.intrSvc.GetByIds(ctx, "question", ids)
		// 这个数据查询不到也不需要担心
		if err1 != nil {
			h.logger.Error("查询数据的点赞数据失败",
				elog.Any("ids", ids),
				elog.FieldErr(err))
		}
	}

	// 获得数据
	return ginx.Result{
		// 在 C 端是下拉刷新
		Data: slice.Map(data, func(idx int, src domain.Question) Question {
			return newQuestion(src, intrs[src.Id])
		}),
	}, nil
}
