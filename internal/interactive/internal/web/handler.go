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
	"github.com/ecodeclub/webook/internal/interactive/internal/domain"
	"github.com/ecodeclub/webook/internal/interactive/internal/service"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc service.InteractiveService
}

func NewHandler(svc service.InteractiveService) *Handler {
	return &Handler{
		svc: svc,
	}
}

// PrivateRoutes 这边我们直接让前端来控制 biz 和 biz_id，简化实现
// 这算是一种反范式的设计和实现方式
func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/intr")
	g.POST("/collect/toggle", ginx.BS[CollectReq](h.Collect))
	g.POST("/like/toggle", ginx.BS[LikeReq](h.Like))
	g.POST("/view", ginx.B[ViewReq](h.View))
	// 获得某个数据的点赞数据
	g.POST("/cnt", ginx.BS[GetCntReq](h.GetCnt))
	g.POST("/detail", ginx.B[BatchGetCntReq](h.BatchGetCnt))
}

func (h *Handler) PublicRoutes(server *gin.Engine) {

}

func (h *Handler) Collect(ctx *ginx.Context, req CollectReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	err := h.svc.CollectToggle(ctx, req.Biz, req.BizId, uid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) GetCnt(ctx *ginx.Context, req GetCntReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	intr, err := h.svc.Get(ctx, req.Biz, req.BizId, uid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: &GetCntResp{
			LikeCnt:    intr.LikeCnt,
			ViewCnt:    intr.ViewCnt,
			Collected:  intr.Collected,
			CollectCnt: intr.CollectCnt,
			Liked:      intr.Liked,
		},
	}, nil
}

func (h *Handler) Like(ctx *ginx.Context, req LikeReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	err := h.svc.LikeToggle(ctx, req.Biz, req.BizId, uid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) View(ctx *ginx.Context, req ViewReq) (ginx.Result, error) {
	err := h.svc.IncrReadCnt(ctx, req.Biz, req.BizId)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) BatchGetCnt(ctx *ginx.Context, req BatchGetCntReq) (ginx.Result, error) {
	intrs, err := h.svc.GetByIds(ctx, req.Biz, req.BizIds)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: &BatatGetCntResp{
			InteractiveMap: h.getInteractiveMap(intrs),
		},
	}, nil
}

func (h *Handler) getInteractiveMap(intrMap map[int64]domain.Interactive) map[int64]Interactive {
	res := make(map[int64]Interactive, len(intrMap))
	for id, intr := range intrMap {
		res[id] = newInteractive(intr)
	}
	return res
}
