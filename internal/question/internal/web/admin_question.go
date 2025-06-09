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
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
)

// AdminHandler 制作库
type AdminHandler struct {
	AdminBaseHandler
	svc               service.Service
	searchSyncService service.SearchSyncService
}

func NewAdminHandler(svc service.Service,searchSvc service.SearchSyncService) *AdminHandler {
	return &AdminHandler{
		svc: svc,
		searchSyncService: searchSvc,
	}
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	server.POST("/question/save", ginx.BS[SaveReq](h.Save))
	server.POST("/question/list", ginx.B[Page](h.List))
	server.POST("/question/detail", ginx.B[Qid](h.Detail))
	server.POST("/question/delete", ginx.B[Qid](h.Delete))
	server.POST("/question/publish", ginx.BS[SaveReq](h.Publish))
	server.GET("/question/search/syncAll", ginx.W(h.SearchSync))
}

func (h *AdminHandler) SearchSync(ctx *ginx.Context) (ginx.Result, error) {
	go h.searchSyncService.SyncAll()
	return ginx.Result{}, nil
}

func (h *AdminHandler) Delete(ctx *ginx.Context, qid Qid) (ginx.Result, error) {
	err := h.svc.Delete(ctx, qid.Qid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *AdminHandler) Save(ctx *ginx.Context,
	req SaveReq,
	sess session.Session) (ginx.Result, error) {
	que := req.Question.toDomain()
	que.Uid = sess.Claims().Uid
	id, err := h.svc.Save(ctx, &que)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) Publish(ctx *ginx.Context, req SaveReq, sess session.Session) (ginx.Result, error) {
	que := req.Question.toDomain()
	que.Uid = sess.Claims().Uid
	id, err := h.svc.Publish(ctx, &que)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	data, cnt, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionList(data, cnt),
	}, nil
}

func (h *AdminHandler) Detail(ctx *ginx.Context, req Qid) (ginx.Result, error) {
	detail, err := h.svc.Detail(ctx, req.Qid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newQuestion(detail, interactive.Interactive{}),
	}, err
}
