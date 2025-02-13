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
	"github.com/ecodeclub/webook/internal/interactive"
	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type QuestionSetHandler struct {
	svc        service.QuestionSetService
	examineSvc service.ExamineService
	logger     *elog.Component
	intrSvc    interactive.Service
	sp         session.Provider
}

func NewQuestionSetHandler(
	svc service.QuestionSetService,
	examineSvc service.ExamineService,
	intrSvc interactive.Service,
	sp session.Provider,
) *QuestionSetHandler {
	return &QuestionSetHandler{
		svc:        svc,
		intrSvc:    intrSvc,
		examineSvc: examineSvc,
		logger:     elog.DefaultLogger,
		sp:         sp,
	}
}

func (h *QuestionSetHandler) PublicRoutes(server *gin.Engine) {
	g := server.Group("/question-sets")
	g.POST("/list", ginx.B[Page](h.ListQuestionSets))
	g.POST("/detail", ginx.B(h.RetrieveQuestionSetDetail))
	g.POST("/detail/biz", ginx.B(h.GetDetailByBiz))
}
func (h *QuestionSetHandler) getUid(gctx *ginx.Context) int64 {
	sess, err := h.sp.Get(gctx)
	if err != nil {
		// 没登录
		return 0
	}
	return sess.Claims().Uid
}

// ListQuestionSets 展示个人题集
func (h *QuestionSetHandler) ListQuestionSets(ctx *ginx.Context, req Page) (ginx.Result, error) {
	uid := h.getUid(ctx)
	data, count, err := h.svc.ListDefault(ctx, req.Offset, req.Limit)
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
		intrs, err1 = h.intrSvc.GetByIds(ctx, "questionSet", uid, ids)
		// 这个数据查询不到也不需要担心
		if err1 != nil {
			h.logger.Error("查询题集的点赞数据失败",
				elog.Any("ids", ids),
				elog.FieldErr(err))
		}
	}
	return ginx.Result{
		Data: QuestionSetList{
			Total: count,
			QuestionSets: slice.Map(data, func(idx int, src domain.QuestionSet) QuestionSet {
				qs := newQuestionSet(src)
				qs.Interactive = newInteractive(intrs[src.Id])
				return qs
			}),
		},
	}, nil
}

func (h *QuestionSetHandler) GetDetailByBiz(
	ctx *ginx.Context,
	req BizReq) (ginx.Result, error) {
	data, err := h.svc.DetailByBiz(ctx, req.Biz, req.BizId)
	if err != nil {
		return systemErrorResult, err
	}
	return h.getDetail(ctx, data)
}

// RetrieveQuestionSetDetail 题集详情
func (h *QuestionSetHandler) RetrieveQuestionSetDetail(
	ctx *ginx.Context,
	req QuestionSetID) (ginx.Result, error) {
	data, err := h.svc.PubDetail(ctx.Request.Context(), req.QSID)
	if err != nil {
		return systemErrorResult, err
	}

	return h.getDetail(ctx, data)
}

func (h *QuestionSetHandler) getDetail(
	ctx *ginx.Context,
	qs domain.QuestionSet) (ginx.Result, error) {
	var (
		eg         errgroup.Group
		intr       interactive.Interactive
		queIntrMap map[int64]interactive.Interactive
		resultMap  map[int64]domain.ExamineResult
		uid        int64
	)
	sess, err := h.sp.Get(ctx)
	if err == nil {
		uid = sess.Claims().Uid
	}

	eg.Go(func() error {
		var err error
		intr, err = h.intrSvc.Get(ctx, "questionSet", qs.Id, uid)
		return err
	})

	eg.Go(func() error {
		var eerr error
		queIntrMap, eerr = h.intrSvc.GetByIds(ctx, "question", uid, qs.Qids())
		return eerr
	})

	eg.Go(func() error {
		var err error
		resultMap, err = h.examineSvc.GetResults(ctx, uid, qs.Qids())
		return err
	})

	err = eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: h.toQuestionSetVO(qs, intr, resultMap, queIntrMap),
	}, nil
}

func (h *QuestionSetHandler) toQuestionSetVO(
	set domain.QuestionSet,
	intr interactive.Interactive,
	results map[int64]domain.ExamineResult,
	queIntrMap map[int64]interactive.Interactive,
) QuestionSet {
	qs := newQuestionSet(set)
	qs.Questions = h.toQuestionVO(set.Questions, results, queIntrMap)
	qs.Interactive = newInteractive(intr)
	return qs
}

func (h *QuestionSetHandler) toQuestionVO(
	questions []domain.Question,
	results map[int64]domain.ExamineResult,
	queIntrMap map[int64]interactive.Interactive) []Question {
	return slice.Map(questions, func(idx int, src domain.Question) Question {
		intr := queIntrMap[src.Id]
		que := newQuestion(src, intr)
		res := results[que.Id]
		que.ExamineResult = res.Result.ToUint8()
		return que
	})
}
