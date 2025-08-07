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
	"fmt"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/comment/internal/domain"
	"github.com/ecodeclub/webook/internal/comment/internal/service"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc service.CommentService
}

func NewHandler(svc service.CommentService) *Handler {
	return &Handler{
		svc: svc,
	}
}

func (h *Handler) PrivateRoutes(_ *gin.Engine) {}

func (h *Handler) PublicRoutes(_ *gin.Engine) {}

func (h *Handler) MemberRoutes(server *gin.Engine) {
	// 在这里注册路由
	group := server.Group("/comment")
	group.POST("/", ginx.BS[CreateRequest](h.Create))
	// 查询直接（始祖）评论，目前按照评论时间的倒序（注意和replies接口的区别）排序
	group.POST("/list", ginx.BS[ListRequest](h.List))
	// 获得某个直接（始祖）评论的所有子评论，孙子评论，按照评论时间倒序排序（即后评论的在前面）
	group.POST("/replies", ginx.BS[RepliesRequest](h.Replies))
	group.POST("/delete", ginx.BS[DeleteRequest](h.Delete))
}

func (h *Handler) Create(ctx *ginx.Context, req CreateRequest, sess session.Session) (ginx.Result, error) {
	if req.Comment.Content == "" {
		return systemErrorResult, errors.New("评论内容不能为空")
	}
	id, err := h.svc.Create(ctx.Request.Context(),
		domain.Comment{
			User: domain.User{
				ID: sess.Claims().Uid,
			},
			Biz:      req.Comment.Biz,
			BizID:    req.Comment.BizID,
			ParentID: req.Comment.ParentID,
			Content:  req.Comment.Content,
			Utime:    req.Comment.Utime,
		})
	if err != nil {
		return systemErrorResult, err
	}
	// 返回评论 ID
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) List(ctx *ginx.Context, req ListRequest, _ session.Session) (ginx.Result, error) {
	ancestors, total, err := h.svc.List(ctx.Request.Context(), req.Biz, req.BizID, req.MinID, req.Limit)
	if err != nil {
		return systemErrorResult, fmt.Errorf("查找%q业务的%d资源的直接评论（始祖评论）失败: %w", req.Biz, req.BizID, err)
	}
	return ginx.Result{
		Data: CommentList{
			List: slice.Map(ancestors, func(_ int, src domain.Comment) Comment {
				return h.toVO(src)
			}),
			Total: int(total),
		},
	}, nil
}

func (h *Handler) toVO(c domain.Comment) Comment {
	return Comment{
		ID: c.ID,
		User: User{
			ID:       c.User.ID,
			Nickname: c.User.NickName,
			Avatar:   c.User.Avatar,
		},
		Biz:        c.Biz,
		BizID:      c.BizID,
		ParentID:   c.ParentID,
		Content:    c.Content,
		Utime:      c.Utime,
		ReplyCount: c.ReplyCount,
	}
}

func (h *Handler) Replies(ctx *ginx.Context, req RepliesRequest, _ session.Session) (ginx.Result, error) {
	descendants, total, err := h.svc.Replies(ctx.Request.Context(), req.AncestorID, req.MinID, req.Limit)
	if err != nil {
		return systemErrorResult, fmt.Errorf("查找评论ID=%d的后裔评论失败: %w", req.AncestorID, err)
	}
	return ginx.Result{
		Data: CommentList{
			List: slice.Map(descendants, func(_ int, src domain.Comment) Comment {
				return h.toVO(src)
			}),
			Total: int(total),
		},
	}, nil
}

func (h *Handler) Delete(ctx *ginx.Context, req DeleteRequest, sess session.Session) (ginx.Result, error) {
	err := h.svc.Delete(ctx.Request.Context(), req.ID, sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Msg: "OK",
	}, nil
}
