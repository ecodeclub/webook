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
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
)

type AdminQuestionSetHandler struct {
	AdminBaseHandler
	svc service.QuestionSetService
}

func NewAdminQuestionSetHandler(svc service.QuestionSetService) *AdminQuestionSetHandler {
	return &AdminQuestionSetHandler{svc: svc}
}

func (h *AdminQuestionSetHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/question-sets")
	g.POST("/save", ginx.BS[QuestionSet](h.SaveQuestionSet))
	g.POST("/questions/save", ginx.BS[UpdateQuestions](h.UpdateQuestions))
	g.POST("/list", ginx.B[Page](h.ListQuestionSets))
	g.POST("/detail", ginx.B(h.RetrieveQuestionSetDetail))
	g.POST("/candidate", ginx.B[CandidateReq](h.Candidate))
}

func (h *AdminQuestionSetHandler) Candidate(ctx *ginx.Context, req CandidateReq) (ginx.Result, error) {
	data, cnt, err := h.svc.GetCandidates(ctx, req.QSID, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionList(data, cnt),
	}, nil
}

// UpdateQuestions 整体更新题集中的所有问题 覆盖式的 前端传递过来的问题集合就是题集中最终的问题集合
func (h *AdminQuestionSetHandler) UpdateQuestions(ctx *ginx.Context, req UpdateQuestions, sess session.Session) (ginx.Result, error) {
	questions := make([]domain.Question, len(req.QIDs))
	for i := range req.QIDs {
		questions[i] = domain.Question{Id: req.QIDs[i]}
	}
	err := h.svc.UpdateQuestions(ctx.Request.Context(), domain.QuestionSet{
		Id:        req.QSID,
		Uid:       sess.Claims().Uid,
		Questions: questions,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

// SaveQuestionSet 保存
func (h *AdminQuestionSetHandler) SaveQuestionSet(ctx *ginx.Context, req QuestionSet, sess session.Session) (ginx.Result, error) {
	id, err := h.svc.Save(ctx.Request.Context(), domain.QuestionSet{
		Id:          req.Id,
		Uid:         sess.Claims().Uid,
		Biz:         req.Biz,
		BizId:       req.BizId,
		Title:       req.Title,
		Description: req.Description,
		Utime:       time.Now(),
	})

	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

// ListQuestionSets 展示个人题集
func (h *AdminQuestionSetHandler) ListQuestionSets(ctx *ginx.Context, req Page) (ginx.Result, error) {
	data, total, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: QuestionSetList{
			Total: total,
			QuestionSets: slice.Map(data, func(idx int, src domain.QuestionSet) QuestionSet {
				qs := newQuestionSet(src)
				return qs
			}),
		},
	}, nil
}

func (h *AdminQuestionSetHandler) RetrieveQuestionSetDetail(
	ctx *ginx.Context,
	req QuestionSetID) (ginx.Result, error) {
	data, err := h.svc.Detail(ctx.Request.Context(), req.QSID)
	if err != nil {
		return systemErrorResult, err
	}
	set := newQuestionSet(data)
	set.Questions = slice.Map(data.Questions, func(idx int, src domain.Question) Question {
		return newQuestion(src, interactive.Interactive{})
	})
	return ginx.Result{
		Data: set,
	}, nil
}
