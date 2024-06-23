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
	"github.com/ecodeclub/webook/internal/search/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc    service.SearchService
	logger *elog.Component
}

func NewHandler(svc service.SearchService) *Handler {
	return &Handler{
		svc:    svc,
		logger: elog.DefaultLogger,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/search/list", ginx.B[SearchReq](h.List))
}

func (h *Handler) List(ctx *ginx.Context, req SearchReq) (ginx.Result, error) {
	data, err := h.svc.Search(ctx, req.Offset, req.Limit, req.Keywords)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: NewSearchResult(data),
	}, nil
}
