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
	"github.com/ecodeclub/webook/internal/label/internal/domain"
	"github.com/ecodeclub/webook/internal/label/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/label")
	g.GET("/system", ginx.W(h.SystemLabels))
	g.POST("/system/create", ginx.B(h.CreateSystemLabel))
}

func (h *Handler) SystemLabels(ctx *ginx.Context) (ginx.Result, error) {
	labels, err := h.svc.SystemLabels(ctx)
	if err != nil {
		return ginx.Result{}, err
	}
	return ginx.Result{
		Data: slice.Map(labels, func(idx int, src domain.Label) Label {
			return Label{
				Id:   src.Id,
				Name: src.Name,
			}
		}),
	}, nil
}

func (h *Handler) CreateSystemLabel(ctx *ginx.Context, req Label) (ginx.Result, error) {
	id, err := h.svc.CreateSystemLabel(ctx, req.Name)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{Data: id}, nil
}
