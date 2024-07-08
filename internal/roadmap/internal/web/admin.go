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
	"github.com/ecodeclub/webook/internal/roadmap/internal/domain"
	"github.com/ecodeclub/webook/internal/roadmap/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	svc    service.AdminService
	bizSvc service.BizService
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/roadmap")
	g.POST("/save", ginx.B(h.Save))
	g.POST("/list", ginx.B(h.List))
	g.POST("/detail", ginx.B(h.Detail))

	edge := g.Group("/edge")
	edge.POST("/save", ginx.B(h.AddEdge))
	edge.POST("/delete", ginx.B(h.DeleteEdge))
}

func (h *AdminHandler) Save(ctx *ginx.Context, req Roadmap) (ginx.Result, error) {
	id, err := h.svc.Save(ctx, req.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	rs, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	bizs := make([]string, 0, len(rs))
	bizIds := make([]int64, 0, len(rs))
	for _, r := range rs {
		bizs = append(bizs, r.Biz)
		bizIds = append(bizIds, r.BizId)
	}
	// 获取 biz 对应的信息
	bizsMap, err := h.bizSvc.GetBizs(ctx, bizs, bizIds)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: RoadmapListResp{
			Total: len(rs),
			Maps: slice.Map(rs, func(idx int, src domain.Roadmap) Roadmap {
				res := newRoadmap(src)
				res.BizTitle = bizsMap[src.Biz][src.BizId].Title
				return res
			}),
		},
	}, nil
}

// AddEdge 后面可以考虑重构为 Save 语义
func (h *AdminHandler) AddEdge(ctx *ginx.Context, req AddEdgeReq) (ginx.Result, error) {
	err := h.svc.AddEdge(ctx, req.Rid, req.Edge.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *AdminHandler) DeleteEdge(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	err := h.svc.DeleteEdge(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *AdminHandler) Detail(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	r, err := h.svc.Detail(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	bizs, bizIds := r.Bizs()
	bizMap, err := h.bizSvc.GetBizs(ctx, bizs, bizIds)
	if err != nil {
		return systemErrorResult, err
	}

	rm := newRoadmapWithBiz(r, bizMap)
	return ginx.Result{
		Data: rm,
	}, nil
}

func NewAdminHandler(
	svc service.AdminService,
	bizSvc service.BizService) *AdminHandler {
	return &AdminHandler{
		svc:    svc,
		bizSvc: bizSvc,
	}
}
