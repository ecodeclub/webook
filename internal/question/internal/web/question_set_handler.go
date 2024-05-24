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

	"github.com/ecodeclub/webook/internal/interactive"
	"golang.org/x/sync/errgroup"

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

var _ ginx.Handler = (*QuestionSetHandler)(nil)

type QuestionSetHandler struct {
	svc     service.QuestionSetService
	logger  *elog.Component
	intrSvc interactive.Service
}

func NewQuestionSetHandler(svc service.QuestionSetService,
	intrSvc interactive.Service) *QuestionSetHandler {
	return &QuestionSetHandler{
		svc:     svc,
		intrSvc: intrSvc,
		logger:  elog.DefaultLogger,
	}
}

func (h *QuestionSetHandler) PublicRoutes(server *gin.Engine) {}

func (h *QuestionSetHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/question-sets")
	g.POST("/save", ginx.BS[SaveQuestionSetReq](h.SaveQuestionSet))
	g.POST("/questions/save", ginx.BS[UpdateQuestionsOfQuestionSetReq](h.UpdateQuestionsOfQuestionSet))
	g.POST("/list", ginx.B[Page](h.ListQuestionSets))
	g.POST("/detail", ginx.BS(h.RetrieveQuestionSetDetail))
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

// ListQuestionSets 展示个人题集
func (h *QuestionSetHandler) ListQuestionSets(ctx *ginx.Context, req Page) (ginx.Result, error) {
	data, total, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	// 查询点赞收藏记录
	intrs := map[int64]interactive.Interactive{}
	if len(data) > 0 {
		ids := slice.Map(data, func(idx int, src domain.QuestionSet) int64 {
			return src.Id
		})
		var err1 error
		intrs, err1 = h.intrSvc.GetByIds(ctx, "questionSet", ids)
		// 这个数据查询不到也不需要担心
		if err1 != nil {
			h.logger.Error("查询题集的点赞数据失败",
				elog.Any("ids", ids),
				elog.FieldErr(err))
		}
	}
	return ginx.Result{
		Data: QuestionSetList{
			Total: total,
			QuestionSets: slice.Map(data, func(idx int, src domain.QuestionSet) QuestionSet {
				return QuestionSet{
					Id:          src.Id,
					Title:       src.Title,
					Description: src.Description,
					Utime:       src.Utime.UnixMilli(),
					Interactive: newInteractive(intrs[src.Id]),
				}
			}),
		},
	}, nil
}

// RetrieveQuestionSetDetail 题集详情
func (h *QuestionSetHandler) RetrieveQuestionSetDetail(
	ctx *ginx.Context,
	req QuestionSetID, sess session.Session) (ginx.Result, error) {
	var (
		eg   errgroup.Group
		data domain.QuestionSet
		intr interactive.Interactive
	)
	eg.Go(func() error {
		var err error
		data, err = h.svc.Detail(ctx.Request.Context(), req.QSID)
		return err
	})

	eg.Go(func() error {
		var err error
		intr, err = h.intrSvc.Get(ctx, "questionSet", req.QSID, sess.Claims().Uid)
		return err
	})

	err := eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionSetVO(data, intr),
	}, nil
}

func (h *QuestionSetHandler) toQuestionSetVO(
	set domain.QuestionSet,
	intr interactive.Interactive) QuestionSet {
	return QuestionSet{
		Id:          set.Id,
		Title:       set.Title,
		Description: set.Description,
		Questions:   h.toQuestionVO(set.Questions),
		Utime:       set.Utime.UnixMilli(),
		Interactive: newInteractive(intr),
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
