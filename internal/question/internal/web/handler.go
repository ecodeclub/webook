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
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
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

func (h *Handler) PublicRoutes(server *gin.Engine) {
	server.POST("/question/pub/list", ginx.B[Page](h.PubList))
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/question/save", ginx.S(h.Permission), ginx.BS[SaveReq](h.Save))
	server.POST("/question/list", ginx.S(h.Permission), ginx.B[Page](h.List))
	server.POST("/question/detail", ginx.S(h.Permission), ginx.B[Qid](h.Detail))
	server.POST("/question/publish", ginx.S(h.Permission), ginx.BS[SaveReq](h.Publish))
}

func (h *Handler) MemberRoutes(server *gin.Engine) {
	server.POST("/question/pub/detail", ginx.B[Qid](h.PubDetail))
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
	data, cnt, err := h.svc.PubList(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionList(data, cnt),
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
				Status:  int32(src.Status),
				Utime:   src.Utime.Format(time.DateTime),
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
		Data: newQuestion(detail),
	}, err
}

func (h *Handler) PubDetail(ctx *ginx.Context, req Qid) (ginx.Result, error) {
	detail, err := h.svc.PubDetail(ctx, req.Qid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newQuestion(detail),
	}, err
}

func (h *Handler) Permission(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	if sess.Claims().Get("creator").StringOrDefault("") != "true" {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return ginx.Result{}, fmt.Errorf("非法访问创作中心 uid: %d", sess.Claims().Uid)
	}
	return ginx.Result{}, ginx.ErrNoResponse
}
