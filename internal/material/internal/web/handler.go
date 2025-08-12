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
	"github.com/ecodeclub/webook/internal/material/internal/domain"
	"github.com/ecodeclub/webook/internal/material/internal/service"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc service.MaterialService
}

func NewHandler(svc service.MaterialService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/material")
	g.POST("/submit", ginx.BS[SubmitMaterialReq](h.Submit))
	g.POST("/history", ginx.BS[ListMaterialsReq](h.History))
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {}

func (h *Handler) Submit(ctx *ginx.Context, req SubmitMaterialReq, sess session.Session) (ginx.Result, error) {
	_, err := h.svc.Submit(ctx.Request.Context(), domain.Material{
		Uid:       sess.Claims().Uid,
		Title:     req.Material.Title,
		AudioURL:  req.Material.AudioURL,
		ResumeURL: req.Material.ResumeURL,
		Remark:    req.Material.Remark,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{Msg: "OK"}, nil
}

func (h *Handler) History(ctx *ginx.Context, req ListMaterialsReq, sess session.Session) (ginx.Result, error) {
	materials, total, err := h.svc.List(ctx.Request.Context(), sess.Claims().Uid, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, fmt.Errorf("获取素材列表失败: %w", err)
	}
	return ginx.Result{
		Data: ListMaterialsResp{
			Total: int(total),
			List: slice.Map(materials, func(idx int, src domain.Material) Material {
				return Material{
					ID:        src.ID,
					Title:     src.Title,
					AudioURL:  src.AudioURL,
					ResumeURL: src.ResumeURL,
					Remark:    src.Remark,
					Status:    src.Status.String(),
					Ctime:     src.Ctime,
					Utime:     src.Utime,
				}
			}),
		},
	}, nil
}
