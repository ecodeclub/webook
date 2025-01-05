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
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc        service.Service
	intrSvc    interactive.Service
	examineSvc service.ExamineService
	logger     *elog.Component
}

func NewHandler(svc service.Service,
	examineSvc service.ExamineService,
	intrSvc interactive.Service) *Handler {
	return &Handler{
		svc:        svc,
		intrSvc:    intrSvc,
		examineSvc: examineSvc,
		logger:     elog.DefaultLogger,
	}
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	server.POST("/case/pub/list", ginx.B[Page](h.PubList))
	server.POST("/cases/list", ginx.B[Page](h.PubList))
}

func (h *Handler) MemberRoutes(server *gin.Engine) {
	server.POST("/cases/detail", ginx.BS(h.PubDetail))
	server.POST("/case/detail", ginx.BS(h.PubDetail))
	server.POST("/case/pub/detail", ginx.BS(h.PubDetail))
}

func (h *Handler) PubList(ctx *ginx.Context, req Page) (ginx.Result, error) {
	count, data, err := h.svc.PubList(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}

	intrs := map[int64]interactive.Interactive{}
	if len(data) > 0 {
		ids := slice.Map(data, func(idx int, src domain.Case) int64 {
			return src.Id
		})
		var err1 error
		intrs, err1 = h.intrSvc.GetByIds(ctx, "case", ids)
		// 这个数据查询不到也不需要担心
		if err1 != nil {
			h.logger.Error("查询数据的点赞数据失败",
				elog.Any("ids", ids),
				elog.FieldErr(err))
		}
	}
	return ginx.Result{
		Data: CasesList{
			Total: count,
			Cases: slice.Map(data, func(idx int, ca domain.Case) Case {
				return Case{
					Id:           ca.Id,
					Title:        ca.Title,
					Introduction: ca.Introduction,
					Labels:       ca.Labels,
					Utime:        ca.Utime.UnixMilli(),
					Interactive:  newInteractive(intrs[ca.Id]),
				}
			}),
		},
	}, nil
}

func (h *Handler) PubDetail(ctx *ginx.Context, req CaseId, sess session.Session) (ginx.Result, error) {
	var (
		eg         errgroup.Group
		detail     domain.Case
		intr       interactive.Interactive
		exmaineRes domain.CaseResult
	)

	uid := sess.Claims().Uid
	eg.Go(func() error {
		var err error
		detail, err = h.svc.PubDetail(ctx, req.Cid)
		return err
	})

	eg.Go(func() error {
		var err error
		intr, err = h.intrSvc.Get(ctx, domain.BizCase, req.Cid, uid)
		return err
	})

	eg.Go(func() error {
		var err error
		exmaineRes, err = h.examineSvc.GetResult(ctx, uid, req.Cid)
		return err
	})

	err := eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}
	res := newCase(detail)
	res.Interactive = newInteractive(intr)
	res.ExamineResult = exmaineRes.ToUint8()
	return ginx.Result{
		Data: res,
	}, err
}

func newCase(ca domain.Case) Case {
	return Case{
		Id:           ca.Id,
		Title:        ca.Title,
		Introduction: ca.Introduction,
		Content:      ca.Content,
		Labels:       ca.Labels,
		GiteeRepo:    ca.GiteeRepo,
		GithubRepo:   ca.GithubRepo,
		Keywords:     ca.Keywords,
		Shorthand:    ca.Shorthand,
		Highlight:    ca.Highlight,
		Guidance:     ca.Guidance,
		Biz:          ca.Biz,
		BizId:        ca.BizId,
		Status:       ca.Status.ToUint8(),
		Utime:        ca.Utime.UnixMilli(),
	}
}
