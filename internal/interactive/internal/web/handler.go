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

var _ ginx.Handler = &Handler{}

type Handler struct {
}

// PrivateRoutes 这边我们直接让前端来控制 biz 和 biz_id，简化实现
// 这算是一种反范式的设计和实现方式
func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/intr")
	g.POST("/collect", ginx.BS[CollectReq](h.Collect))
	g.POST("/like", ginx.BS[LikeReq](h.Like))
	// 统一用 POST 请求，懒得去处理不同的
	g.POST("/cnt", ginx.BS[GetCntReq](h.GetCnt))
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	//TODO implement me
	panic("implement me")
}

func (h *Handler) Collect(ctx *ginx.Context, req CollectReq, sess session.Session) (ginx.Result, error) {
	return ginx.Result{Msg: "OK"}, nil
}

func (h *Handler) GetCnt(ctx *ginx.Context, req GetCntReq, sess session.Session) (ginx.Result, error) {
	return ginx.Result{
		Data: &GetCntResp{},
	}, nil
}

func (h *Handler) Like(ctx *ginx.Context, req LikeReq, sess session.Session) (ginx.Result, error) {
	return ginx.Result{Msg: "OK"}, nil
}
