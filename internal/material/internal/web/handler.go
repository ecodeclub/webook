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
	"fmt"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/material/internal/domain"
	"github.com/ecodeclub/webook/internal/material/internal/event"
	"github.com/ecodeclub/webook/internal/material/internal/service"
	notificationevt "github.com/ecodeclub/webook/internal/notification/event"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc      service.MaterialService
	producer event.WechatRobotEventProducer
	logger   *elog.Component
}

func NewHandler(
	svc service.MaterialService,
	producer event.WechatRobotEventProducer,
) *Handler {
	return &Handler{
		svc:      svc,
		producer: producer,
		logger:   elog.DefaultLogger.With(elog.FieldComponentName("material.Handler")),
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/material")
	g.POST("/list", ginx.BS[ListMaterialsReq](h.List))
	g.POST("/save", ginx.BS[SaveMaterialReq](h.Save))
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {}

func (h *Handler) Save(ctx *ginx.Context, req SaveMaterialReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	id, err := h.svc.Submit(ctx.Request.Context(), domain.Material{
		Uid:       uid,
		Title:     req.Material.Title,
		AudioURL:  req.Material.AudioURL,
		ResumeURL: req.Material.ResumeURL,
		Remark:    req.Material.Remark,
	})
	if err != nil {
		return systemErrorResult, err
	}
	evt := notificationevt.WechatRobotEvent{
		Robot:      "adminRobot",
		RawContent: fmt.Sprintf("用户%d刚刚提交了新素材%q（%d）请前往后台查看！", uid, req.Material.Title, id),
	}
	if er := h.producer.Produce(ctx.Request.Context(), evt); er != nil {
		h.logger.Error("发送企业微信群通知失败",
			elog.FieldErr(er),
			elog.Any("event", evt),
		)
	}
	return ginx.Result{Msg: "OK"}, nil
}

func (h *Handler) List(ctx *ginx.Context, req ListMaterialsReq, sess session.Session) (ginx.Result, error) {
	materials, total, err := h.svc.List(ctx.Request.Context(), sess.Claims().Uid, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, fmt.Errorf("获取素材列表失败: %w", err)
	}
	return ginx.Result{
		Data: ListMaterialsResp{
			Total: int(total),
			List: slice.Map(materials, func(idx int, src domain.Material) Material {
				return Material{
					ID:        src.ID,
					Title:     src.Title,
					AudioURL:  src.AudioURL,
					ResumeURL: src.ResumeURL,
					Remark:    src.Remark,
					Status:    src.Status.String(),
					Ctime:     src.Ctime,
					Utime:     src.Utime,
				}
			}),
		},
	}, nil
}
