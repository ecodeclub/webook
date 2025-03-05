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
	"errors"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/errs"
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
	// 暂时保留一年，因为后期 AI 的变化难以预测
	g.POST("", ginx.BS(h.Examine))
	// 觉得 AI 的评价不准确，那么可以调用这个接口来修正，这是直接暴露给用户使用的
	g.POST("/correct", ginx.BS[CorrectReq](h.Correct))
}

func (h *ExamineHandler) Examine(ctx *ginx.Context, req ExamineReq, sess session.Session) (ginx.Result, error) {
	res, err := h.svc.Examine(ctx, sess.Claims().Uid, req.Qid, req.Input)
	switch {
	case errors.Is(err, service.ErrInsufficientCredit):
		return ginx.Result{
			Code: errs.InsufficientCredit.Code,
			Msg:  errs.InsufficientCredit.Msg,
		}, nil

	case err == nil:
		return ginx.Result{
			Data: newExamineResult(res),
		}, nil
	default:
		return systemErrorResult, err
	}
}

// Correct 修改题目的结果
func (h *ExamineHandler) Correct(ctx *ginx.Context, req CorrectReq, sess session.Session) (ginx.Result, error) {
	// 实现这个接口
	err := h.svc.Correct(ctx, sess.Claims().Uid, req.Qid, domain.Result(req.Result))
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newExamineResult(domain.ExamineResult{
			Qid:    req.Qid,
			Result: domain.Result(req.Result),
		}),
	}, nil
}
