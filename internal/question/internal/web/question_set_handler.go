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

	"github.com/ecodeclub/ekit/bean/copier"
	"github.com/ecodeclub/ekit/bean/copier/converter"
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type QuestionSetHandler struct {
	dm2vo  copier.Copier[domain.QuestionSet, QuestionSet]
	svc    service.QuestionSetService
	logger *elog.Component
}

func NewQuestionSetHandler(svc service.QuestionSetService) (*QuestionSetHandler, error) {
	dm2vo, err := copier.NewReflectCopier[domain.QuestionSet, QuestionSet](
		copier.ConvertField[time.Time, string]("Utime", converter.ConverterFunc[time.Time, string](func(src time.Time) (string, error) {
			return src.Format(time.DateTime), nil
		})),
	)
	if err != nil {
		return nil, err
	}
	return &QuestionSetHandler{
		dm2vo:  dm2vo,
		svc:    svc,
		logger: elog.DefaultLogger,
	}, nil
}

func (h *QuestionSetHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/question-sets")
	g.POST("/save", ginx.BS[SaveQuestionSetReq](h.SaveQuestionSet))
	g.POST("/questions/save", ginx.BS[UpdateQuestionsOfQuestionSetReq](h.UpdateQuestionsOfQuestionSet))
	g.POST("/list", ginx.B[Page](h.ListPrivateQuestionSets))
	g.POST("/detail", ginx.B[QuestionSetID](h.RetrieveQuestionSetDetail))

	g.POST("/pub/list", ginx.B[Page](h.ListAllQuestionSets))
}

// SaveQuestionSet 保存
func (h *QuestionSetHandler) SaveQuestionSet(ctx *ginx.Context, req SaveQuestionSetReq, sess session.Session) (ginx.Result, error) {
	id, err := h.svc.Save(ctx.Request.Context(), domain.QuestionSet{
		Id:          req.Id,
		Uid:         sess.Claims().Uid,
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

// UpdateQuestionsOfQuestionSet 整体更新题集中的所有问题 覆盖式的 前端传递过来的问题集合就是题集中最终的问题集合
func (h *QuestionSetHandler) UpdateQuestionsOfQuestionSet(ctx *ginx.Context, req UpdateQuestionsOfQuestionSetReq, sess session.Session) (ginx.Result, error) {
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

// ListPrivateQuestionSets 展示个人题集
func (h *QuestionSetHandler) ListPrivateQuestionSets(ctx *ginx.Context, req Page) (ginx.Result, error) {
	data, total, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionSetList(data, total),
	}, nil
}

func (h *QuestionSetHandler) toQuestionSetList(data []domain.QuestionSet, total int64) QuestionSetList {
	return QuestionSetList{
		Total: total,
		QuestionSets: slice.Map(data, func(idx int, src domain.QuestionSet) QuestionSet {
			dm2vo, _ := copier.NewReflectCopier[domain.QuestionSet, QuestionSet](
				// 忽略题集中的问题列表
				copier.IgnoreFields("Questions"),
				copier.ConvertField[time.Time, string]("Utime", converter.ConverterFunc[time.Time, string](func(src time.Time) (string, error) {
					return src.Format(time.DateTime), nil
				})),
			)
			dst, err := dm2vo.Copy(&src)
			if err != nil {
				h.logger.Error("转化为 vo 失败", elog.FieldErr(err))
				return QuestionSet{}
			}
			return *dst
		}),
	}
}

// ListAllQuestionSets 展示所有题集
func (h *QuestionSetHandler) ListAllQuestionSets(ctx *ginx.Context, req Page) (ginx.Result, error) {
	data, total, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionSetList(data, total),
	}, nil
}

// RetrieveQuestionSetDetail 题集详情
func (h *QuestionSetHandler) RetrieveQuestionSetDetail(ctx *ginx.Context, req QuestionSetID) (ginx.Result, error) {
	data, err := h.svc.Detail(ctx.Request.Context(), req.QSID)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionSetVO(data),
	}, nil
}

func (h *QuestionSetHandler) toQuestionSetVO(set domain.QuestionSet) QuestionSet {
	return QuestionSet{
		Id:          set.Id,
		Title:       set.Title,
		Description: set.Description,
		Questions:   h.toQuestionVO(set.Questions),
		Utime:       set.Utime.Format(time.DateTime),
	}
}

func (h *QuestionSetHandler) toQuestionVO(questions []domain.Question) []Question {
	dm2vo, _ := copier.NewReflectCopier[domain.Question, Question](
		copier.ConvertField[time.Time, string]("Utime", converter.ConverterFunc[time.Time, string](func(src time.Time) (string, error) {
			return src.Format(time.DateTime), nil
		})),
	)
	vos := make([]Question, len(questions))
	for i, question := range questions {
		vo, _ := dm2vo.Copy(&question)
		vos[i] = *vo
	}
	return vos
}
