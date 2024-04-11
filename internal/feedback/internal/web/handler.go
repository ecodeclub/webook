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

func (h *Handler) MemberRoutes(server *gin.Engine) {
	// 列表 根据交互来, 先是未处理，然后是通过，最后是拒绝
	server.POST("/feedback/list", ginx.S(h.Permission), ginx.B(h.List))
	// 未处理的个数
	server.GET("/feedback/pending-count", ginx.S(h.Permission), ginx.W(h.PendingCount))
	server.POST("/feedback/detail", ginx.S(h.Permission), ginx.B(h.Detail))
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
	data, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: FeedbackList{
			Feedbacks: slice.Map(data, func(idx int, feedBack domain.Feedback) Feedback {
				return newFeedback(feedBack)
			}),
		},
	}, nil
}
func (h *Handler) Detail(ctx *ginx.Context, req FeedbackID) (ginx.Result, error) {
	detail, err := h.svc.Info(ctx, req.FID)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newFeedback(detail),
	}, err
}
func (h *Handler) UpdateStatus(ctx *ginx.Context, req UpdateStatusReq) (ginx.Result, error) {
	err := h.svc.UpdateStatus(ctx, domain.Feedback{
		ID:     req.FID,
		Status: domain.FeedbackStatus(req.Status),
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, err
}

func (h *Handler) Create(ctx *ginx.Context, req CreateReq, sess session.Session) (ginx.Result, error) {
	feedBack := req.Feedback.toDomain()
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
