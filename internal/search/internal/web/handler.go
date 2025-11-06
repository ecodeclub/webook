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
	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/sync/errgroup"
)

type Handler struct {
	svc     service.SearchService
	logger  *elog.Component
	examSvc cases.ExamineService
	intrSvc interactive.Service
}

func NewHandler(svc service.SearchService,
	examSvc cases.ExamineService,
	intrSvc interactive.Service,
) *Handler {
	return &Handler{
		svc:     svc,
		logger:  elog.DefaultLogger,
		examSvc: examSvc,
		intrSvc: intrSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/search/list", ginx.BS[SearchReq](h.List))
}

func (h *Handler) List(ctx *ginx.Context, req SearchReq, sess session.Session) (ginx.Result, error) {
	stdCtx := ctx.Request.Context()

	data, err := h.svc.Search(stdCtx, req.Offset, req.Limit, req.Keywords)
	if err != nil {
		return systemErrorResult, err
	}
	var (
		eg              errgroup.Group
		examMap         map[int64]cases.ExamineResult
		questionIntrMap = make(map[int64]interactive.Interactive, len(data.Questions))
		caseIntrMap     = make(map[int64]interactive.Interactive, len(data.Cases))
	)
	uid := sess.Claims().Uid
	cids := slice.Map(data.Cases, func(idx int, src domain.Case) int64 {
		return src.Id
	})
	qids := slice.Map(data.Questions, func(idx int, src domain.Question) int64 {
		return src.ID
	})

	if data.Cases != nil {
		eg.Go(func() error {
			examMap, err = h.examSvc.GetResults(stdCtx, uid, cids)
			return err
		})
	}
	if len(data.Questions) > 0 {
		eg.Go(func() error {
			intrs, err1 := h.intrSvc.GetByIds(stdCtx, "question", uid, qids)
			// 这个数据查询不到也不需要担心
			if err1 != nil {
				h.logger.Error("查询数据的点赞数据失败",
					elog.Any("ids", qids),
					elog.FieldErr(err))
			}
			for idx := range intrs {
				intr := intrs[idx]
				questionIntrMap[intr.BizId] = intr
			}
			return nil
		})
	}
	if len(data.Cases) > 0 {
		eg.Go(func() error {
			intrs, err1 := h.intrSvc.GetByIds(stdCtx, "case", uid, cids)
			// 这个数据查询不到也不需要担心
			if err1 != nil {
				h.logger.Error("查询数据的点赞数据失败",
					elog.Any("ids", cids),
					elog.FieldErr(err))
			}
			for idx := range intrs {
				intr := intrs[idx]
				caseIntrMap[intr.BizId] = intr
			}
			return nil
		})
	}
	err = eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}

	return ginx.Result{
		Data: NewCSearchResult(data, examMap, caseIntrMap, questionIntrMap),
	}, nil
}
