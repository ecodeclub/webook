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
	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ecodeclub/webook/internal/project/internal/service"
	"github.com/gin-gonic/gin"
)

// Handler C 端接口
type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/project")
	g.POST("/list", ginx.B[Page](h.List))
	g.POST("/detail", ginx.B[IdReq](h.Detail))
}

func (h *Handler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	res, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: slice.Map(res, func(idx int, src domain.Project) Project {
			return newProject(src)
		}),
	}, nil
}

func (h *Handler) Detail(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	res, err := h.svc.Detail(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newProject(res),
	}, nil

}
