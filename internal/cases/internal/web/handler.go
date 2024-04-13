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
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
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
	server.POST("/case/pub/list", ginx.B[Page](h.PubList))
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/case/save", ginx.S(h.Permission), ginx.BS[SaveReq](h.Save))
	server.POST("/case/list", ginx.S(h.Permission), ginx.B[Page](h.List))
	server.POST("/case/detail", ginx.S(h.Permission), ginx.B[CaseId](h.Detail))
	server.POST("/case/publish", ginx.S(h.Permission), ginx.BS[SaveReq](h.Publish))
}

func (h *Handler) MemberRoutes(server *gin.Engine) {
	server.POST("/case/pub/detail", ginx.B[CaseId](h.PubDetail))
}

func (h *Handler) Save(ctx *ginx.Context,
	req SaveReq,
	sess session.Session) (ginx.Result, error) {
	ca := req.Case.toDomain()
	ca.Uid = sess.Claims().Uid
	id, err := h.svc.Save(ctx, &ca)
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
		Data: h.toCaseList(data, cnt),
	}, nil
}

func (h *Handler) Detail(ctx *ginx.Context, req CaseId) (ginx.Result, error) {
	detail, err := h.svc.Detail(ctx, req.Cid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newCase(detail),
	}, err
}

func (h *Handler) PubList(ctx *ginx.Context, req Page) (ginx.Result, error) {
	data, cnt, err := h.svc.PubList(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: CasesList{
			Total: cnt,
			Cases: slice.Map(data, func(idx int, ca domain.Case) Case {
				return Case{
					Id:     ca.Id,
					Title:  ca.Title,
					Labels: ca.Labels,
					Utime:  ca.Utime.Format(time.DateTime),
				}
			}),
		},
	}, nil
}

func (h *Handler) PubDetail(ctx *ginx.Context, req CaseId) (ginx.Result, error) {
	detail, err := h.svc.PubDetail(ctx, req.Cid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newCase(detail),
	}, err
}

func (h *Handler) Publish(ctx *ginx.Context, req SaveReq, sess session.Session) (ginx.Result, error) {
	ca := req.Case.toDomain()
	ca.Uid = sess.Claims().Uid
	id, err := h.svc.Publish(ctx, &ca)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) toCaseList(data []domain.Case, cnt int64) CasesList {
	return CasesList{
		Total: cnt,
		Cases: slice.Map(data, func(idx int, ca domain.Case) Case {
			return newCase(ca)
		}),
	}
}

func newCase(ca domain.Case) Case {
	return Case{
		Id:        ca.Id,
		Title:     ca.Title,
		Content:   ca.Content,
		Labels:    ca.Labels,
		CodeRepo:  ca.CodeRepo,
		Keywords:  ca.Keywords,
		Shorthand: ca.Shorthand,
		Highlight: ca.Highlight,
		Guidance:  ca.Guidance,
		Status:    int32(ca.Status),
		Utime:     ca.Utime.Format(time.DateTime),
	}
}

func (h *Handler) Permission(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	if sess.Claims().Get("creator").StringOrDefault("") != "true" {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return ginx.Result{}, fmt.Errorf("非法访问创作中心 uid: %d", sess.Claims().Uid)
	}
	return ginx.Result{}, ginx.ErrNoResponse
}
