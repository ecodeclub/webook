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
	"github.com/ecodeclub/webook/internal/interactive/internal/domain"
	"github.com/ecodeclub/webook/internal/interactive/internal/service"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{
		svc: svc,
	}
}

// PrivateRoutes 这边我们直接让前端来控制 biz 和 biz_id，简化实现
// 这算是一种反范式的设计和实现方式
func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/interactive")
	g.POST("/collect/toggle", ginx.BS[CollectReq](h.Collect))
	// 创建一个收藏夹
	g.POST("/collection/save", ginx.BS[Collection](h.CollectionSave))
	g.POST("/collection/list", ginx.BS[Page](h.CollectionList))
	g.POST("/collection/delete", ginx.BS[IdReq](h.CollectionDelete))
	g.POST("/collection/move", ginx.BS[MoveCollectionReq](h.MoveCollection))

	g.POST("/like/toggle", ginx.BS[LikeReq](h.Like))
}

func (h *Handler) PublicRoutes(server *gin.Engine) {

}

func (h *Handler) Collect(ctx *ginx.Context, req CollectReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	err := h.svc.CollectToggle(ctx.Request.Context(), req.Biz, req.BizId, uid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) Like(ctx *ginx.Context, req LikeReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	err := h.svc.LikeToggle(ctx.Request.Context(), req.Biz, req.BizId, uid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}

func (h *Handler) CollectionSave(ctx *ginx.Context, req Collection, sess session.Session) (ginx.Result, error) {
	// 把 ID 返回回来
	uid := sess.Claims().Uid
	id, err := h.svc.SaveCollection(ctx, domain.Collection{
		Id:   req.Id,
		Name: req.Name,
		Uid:  uid,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) CollectionList(ctx *ginx.Context, req Page, sess session.Session) (ginx.Result, error) {
	// 根据 ID 倒序返回数据
	uid := sess.Claims().Uid
	collections, err := h.svc.CollectionList(ctx, uid, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: slice.Map(collections, func(idx int, src domain.Collection) Collection {
			return Collection{
				Id:   src.Id,
				Name: src.Name,
			}
		}),
	}, nil
}

func (h *Handler) CollectionDelete(ctx *ginx.Context, req IdReq, sess session.Session) (ginx.Result, error) {
	// 删除这个 id 的 collection
	// 要注意， Uid 必须是这个人。也就是说 A 用户不能删了 B 用户的收藏夹
	uid := sess.Claims().Uid
	err := h.svc.DeleteCollection(ctx, uid, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: "OK",
	}, nil
}

func (h *Handler) MoveCollection(ctx *ginx.Context, req MoveCollectionReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	err := h.svc.MoveToCollection(ctx, req.Biz, req.BizId, uid, req.CollectionId)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{}, nil
}
