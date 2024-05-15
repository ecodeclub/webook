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
	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ecodeclub/webook/internal/project/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/sync/errgroup"
)

// Handler C 端接口
type Handler struct {
	svc     service.Service
	intrSvc interactive.Service
	logger  *elog.Component
}

func NewHandler(svc service.Service, intrSvc interactive.Service) *Handler {
	return &Handler{
		svc:     svc,
		intrSvc: intrSvc,
		logger:  elog.DefaultLogger,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/project")
	g.POST("/list", ginx.B[Page](h.List))
	g.POST("/detail", ginx.BS(h.Detail))
}

func (h *Handler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	data, err := h.svc.List(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	// 查询点赞收藏记录
	intrs := map[int64]interactive.Interactive{}
	if len(data) > 0 {
		ids := slice.Map(data, func(idx int, src domain.Project) int64 {
			return src.Id
		})
		var err1 error
		intrs, err1 = h.intrSvc.GetByIds(ctx, "question", ids)
		// 这个数据查询不到也不需要担心
		if err1 != nil {
			h.logger.Error("查询数据的点赞数据失败",
				elog.Any("ids", ids),
				elog.FieldErr(err))
		}
	}
	return ginx.Result{
		Data: slice.Map(data, func(idx int, src domain.Project) Project {
			return newProject(src, intrs[src.Id])
		}),
	}, nil
}

func (h *Handler) Detail(ctx *ginx.Context, req IdReq, sess session.Session) (ginx.Result, error) {
	var (
		eg     errgroup.Group
		detail domain.Project
		intr   interactive.Interactive
	)
	eg.Go(func() error {
		var err error
		detail, err = h.svc.Detail(ctx, req.Id)
		return err
	})
	eg.Go(func() error {
		var err error
		intr, err = h.intrSvc.Get(ctx, domain.BizProject, req.Id, sess.Claims().Uid)
		return err
	})
	err := eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newProject(detail, intr),
	}, nil

}
