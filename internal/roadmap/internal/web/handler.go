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
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/roadmap/internal/service"
	"github.com/ecodeclub/webook/internal/roadmap/internal/service/biz"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc    service.Service
	bizSvc biz.Service
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/roadmap")
	g.POST("/detail", ginx.B(h.Detail))
}

func (h *Handler) Detail(ctx *ginx.Context, req Biz) (ginx.Result, error) {
	r, err := h.svc.Detail(ctx, req.Biz, req.BizId)
	switch err {
	case service.ErrRoadmapNotFound:
		// 没有
		return ginx.Result{}, nil
	case nil:
		bizs, bizIds := r.Bizs()
		bizMap, err := h.bizSvc.GetBizs(ctx, bizs, bizIds)
		if err != nil {
			return systemErrorResult, err
		}

		rm := newRoadmapWithBiz(r, bizMap)
		return ginx.Result{
			Data: rm,
		}, nil
	default:
		return systemErrorResult, err
	}
}

func NewHandler(svc service.Service, bizSvc biz.Service) *Handler {
	return &Handler{
		svc:    svc,
		bizSvc: bizSvc,
	}
}
