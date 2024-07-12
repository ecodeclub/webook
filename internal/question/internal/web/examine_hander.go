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
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
)

type ExamineHandler struct {
	svc service.ExamineService
}

func NewExamineHandler(svc service.ExamineService) *ExamineHandler {
	return &ExamineHandler{
		svc: svc,
	}
}

func (h *ExamineHandler) MemberRoutes(server *gin.Engine) {
	g := server.Group("/question/examine")
	g.POST("", ginx.BS(h.Examine))
}

func (h *ExamineHandler) Examine(ctx *ginx.Context, req ExamineReq, sess session.Session) (ginx.Result, error) {
	res, err := h.svc.Examine(ctx, sess.Claims().Uid, req.Qid, req.Input)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newExamineResult(res),
	}, nil
}
