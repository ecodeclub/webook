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
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminCaseHandler struct {
	svc       service.Service
	searchSvc service.SearchSyncService
}

func NewAdminCaseHandler(svc service.Service, searchSvc service.SearchSyncService) *AdminCaseHandler {
	return &AdminCaseHandler{
		svc:       svc,
		searchSvc: searchSvc,
	}
}

func (h *AdminCaseHandler) PrivateRoutes(server *gin.Engine) {
	server.POST("/cases/save", ginx.BS[SaveReq](h.Save))
	server.POST("/cases/list", ginx.B[Page](h.List))
	server.POST("/cases/detail", ginx.B[CaseId](h.Detail))
	server.POST("/cases/publish", ginx.BS[SaveReq](h.Publish))
	server.GET("/cases/search/syncAll", ginx.W(h.SyncAll))
}
func (h *AdminCaseHandler) SyncAll(ctx *ginx.Context) (ginx.Result, error) {
	go h.searchSvc.SyncAll()
	return ginx.Result{}, nil
}

func (h *AdminCaseHandler) Save(ctx *ginx.Context,
	req SaveReq,
	sess session.Session) (ginx.Result, error) {
	ca := req.Case.toDomain()
	ca.Uid = sess.Claims().Uid
	id, err := h.svc.Save(ctx, ca)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminCaseHandler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	data, cnt, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: CasesList{
			Total: cnt,
			Cases: slice.Map(data, func(idx int, ca domain.Case) Case {
				return newCase(ca)
			}),
		},
	}, nil
}

func (h *AdminCaseHandler) Detail(ctx *ginx.Context, req CaseId) (ginx.Result, error) {
	detail, err := h.svc.Detail(ctx, req.Cid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newCase(detail),
	}, err
}

func (h *AdminCaseHandler) Publish(ctx *ginx.Context, req SaveReq, sess session.Session) (ginx.Result, error) {
	ca := req.Case.toDomain()
	ca.Uid = sess.Claims().Uid
	id, err := h.svc.Publish(ctx, ca)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}
