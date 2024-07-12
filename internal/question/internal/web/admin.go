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
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
)

// AdminHandler 制作库
type AdminHandler struct {
	svc service.Service
}

func NewAdminHandler(svc service.Service) *AdminHandler {
	return &AdminHandler{
		svc: svc,
	}
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	server.POST("/question/save", ginx.BS[SaveReq](h.Save))
	server.POST("/question/list", ginx.B[Page](h.List))
	server.POST("/question/detail", ginx.B[Qid](h.Detail))
	server.POST("/question/delete", ginx.B[Qid](h.Delete))
	server.POST("/question/publish", ginx.BS[SaveReq](h.Publish))
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
	// 制作库不需要统计总数
	data, cnt, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionList(data, cnt),
	}, nil
}

func (h *AdminHandler) toQuestionList(data []domain.Question, cnt int64) QuestionList {
	return QuestionList{
		Total: cnt,
		Questions: slice.Map(data, func(idx int, src domain.Question) Question {
			return Question{
				Id:      src.Id,
				Title:   src.Title,
				Content: src.Content,
				Labels:  src.Labels,
				Status:  src.Status.ToUint8(),
				Utime:   src.Utime.UnixMilli(),
			}
		}),
	}
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
