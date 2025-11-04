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
	"github.com/ecodeclub/webook/internal/kbase/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type AdminHandler struct {
	syncSvc service.SyncService
	logger  *elog.Component
}

func NewAdminHandler(syncSvc service.SyncService) *AdminHandler {
	return &AdminHandler{
		syncSvc: syncSvc,
		logger:  elog.DefaultLogger.With(elog.FieldComponent("kbase.web.AdminHandler")),
	}
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	server.POST("/kbase/sync/upsert", ginx.B(h.Upsert))
	server.POST("/kbase/sync/batch-upsert", ginx.B(h.BatchUpsert))
	server.POST("/kbase/sync/delete", ginx.B(h.Delete))
}

func (h *AdminHandler) Upsert(ctx *ginx.Context, req Req) (ginx.Result, error) {
	err := h.syncSvc.Upsert(ctx, req.Biz, req.BizID)
	if err != nil {
		h.logger.Error("Upsert 失败", elog.FieldErr(err))
		return systemErrorResult, nil
	}
	return ginx.Result{
		Msg: "ok",
	}, nil
}

func (h *AdminHandler) BatchUpsert(ctx *ginx.Context, req BatchUpsertReq) (ginx.Result, error) {
	err := h.syncSvc.UpsertSince(ctx, req.Biz, req.Since)
	if err != nil {
		h.logger.Error("BatchUpsert 失败", elog.FieldErr(err))
		return systemErrorResult, nil
	}
	return ginx.Result{
		Msg: "ok",
	}, nil
}

func (h *AdminHandler) Delete(ctx *ginx.Context, req Req) (ginx.Result, error) {
	err := h.syncSvc.Delete(ctx, req.Biz, req.BizID)
	if err != nil {
		h.logger.Error("Delete 失败", elog.FieldErr(err))
		return systemErrorResult, nil
	}
	return ginx.Result{
		Msg: "ok",
	}, nil
}
