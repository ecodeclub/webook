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
	"net/http"

	"github.com/ecodeclub/webook/internal/interactive"
	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc     service.Service
	logger  *elog.Component
	intrSvc interactive.Service
}

func NewHandler(svc service.Service, intrSvc interactive.Service) *Handler {
	return &Handler{
		svc:     svc,
		intrSvc: intrSvc,
		logger:  elog.DefaultLogger,
	}
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	server.POST("/question/pub/list", ginx.B[Page](h.PubList))
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/question/save", ginx.S(h.Permission), ginx.BS[SaveReq](h.Save))
	server.POST("/question/list", ginx.S(h.Permission), ginx.B[Page](h.List))
	server.POST("/question/detail", ginx.S(h.Permission), ginx.B[Qid](h.Detail))
	server.POST("/question/delete", ginx.S(h.Permission), ginx.B[Qid](h.Delete))
	server.POST("/question/publish", ginx.S(h.Permission), ginx.BS[SaveReq](h.Publish))
}

func (h *Handler) MemberRoutes(server *gin.Engine) {
	server.POST("/question/pub/detail", ginx.BS[Qid](h.PubDetail))
}

func (h *Handler) Delete(ctx *ginx.Context, qid Qid) (ginx.Result, error) {
	err := h.svc.Delete(ctx, qid.Qid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) Save(ctx *ginx.Context,
	req SaveReq,
	sess session.Session) (ginx.Result, error) {
	que := req.Question.toDomain()
	que.Uid = sess.Claims().Uid
	id, err := h.svc.Save(ctx, &que)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) Publish(ctx *ginx.Context, req SaveReq, sess session.Session) (ginx.Result, error) {
	que := req.Question.toDomain()
	que.Uid = sess.Claims().Uid
	id, err := h.svc.Publish(ctx, &que)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	// 制作库不需要统计总数
	data, cnt, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionList(data, cnt),
	}, nil
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
			return Question{
				Id:          src.Id,
				Title:       src.Title,
				Content:     src.Content,
				Labels:      src.Labels,
				Status:      src.Status.ToUint8(),
				Utime:       src.Utime.UnixMilli(),
				Interactive: newInteractive(intrs[src.Id]),
			}
		}),
	}, nil
}

func (h *Handler) toQuestionList(data []domain.Question, cnt int64) QuestionList {
	return QuestionList{
		Total: cnt,
		Questions: slice.Map(data, func(idx int, src domain.Question) Question {
			return Question{
				Id:      src.Id,
				Title:   src.Title,
				Content: src.Content,
				Labels:  src.Labels,
				Status:  src.Status.ToUint8(),
				Utime:   src.Utime.UnixMilli(),
			}
		}),
	}
}

func (h *Handler) Detail(ctx *ginx.Context, req Qid) (ginx.Result, error) {
	detail, err := h.svc.Detail(ctx, req.Qid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newQuestion(detail, interactive.Interactive{}),
	}, err
}

func (h *Handler) PubDetail(ctx *ginx.Context,
	req Qid, sess session.Session) (ginx.Result, error) {
	var (
		eg     errgroup.Group
		detail domain.Question
		intr   interactive.Interactive
	)
	eg.Go(func() error {
		var err error
		detail, err = h.svc.PubDetail(ctx, req.Qid)
		return err
	})

	eg.Go(func() error {
		var err error
		intr, err = h.intrSvc.Get(ctx, domain.QuestionBiz, req.Qid, sess.Claims().Uid)
		return err
	})
	err := eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newQuestion(detail, intr),
	}, err
}

func (h *Handler) Permission(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	if sess.Claims().Get("creator").StringOrDefault("") != "true" {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return ginx.Result{}, fmt.Errorf("非法访问创作中心 uid: %d", sess.Claims().Uid)
	}
	return ginx.Result{}, ginx.ErrNoResponse
}
