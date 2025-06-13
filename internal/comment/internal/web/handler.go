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
	"github.com/gin-gonic/gin"
)

type Handler struct {
}

func (h *Handler) MemberRoutes(server *gin.Engine) {
	// 在这里注册路由
	group := server.Group("/comment")
	group.POST("/", ginx.BS[CommentRequest](h.Comment))
	// 目前按照评论时间的倒序（注意和replies接口的区别）排序
	group.POST("/list", ginx.BS[CommentRequest](h.CommentList))
	// 获得某个评论的所有子评论，孙子评论，按照评论时间排序（即先评论的在前面）
	group.POST("/replies", ginx.BS[GetRepliesRequest](h.GetReplies))
}

func (h *Handler) Comment(ctx *ginx.Context, req CommentRequest, sess session.Session) (ginx.Result, error) {
	// 返回评论 ID
	return ginx.Result{
		Data: 123,
	}, nil
}

func (h *Handler) CommentList(ctx *ginx.Context, req CommentRequest, sess session.Session) (ginx.Result, error) {
	// 查询某个业务下的评论，直接评论，
	return ginx.Result{
		Data: CommentList{},
	}, nil
}

func (h *Handler) GetReplies(ctx *ginx.Context, req GetRepliesRequest, sess session.Session) (ginx.Result, error) {
	return ginx.Result{
		Data: CommentList{},
	}, nil
}
