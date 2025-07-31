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
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/material/internal/domain"
	"github.com/ecodeclub/webook/internal/material/internal/event"
	"github.com/ecodeclub/webook/internal/material/internal/service"
	sms "github.com/ecodeclub/webook/internal/sms/client"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type AdminHandler struct {
	svc       service.MaterialService
	userSvc   user.UserService
	producer  event.MemberEventProducer
	smsClient sms.Client
	logger    *elog.Component
}

func NewAdminHandler(svc service.MaterialService,
	userSvc user.UserService,
	producer event.MemberEventProducer,
	smsClient sms.Client) *AdminHandler {
	return &AdminHandler{
		svc:       svc,
		userSvc:   userSvc,
		producer:  producer,
		smsClient: smsClient,
		logger:    elog.DefaultLogger.With(elog.FieldComponentName("material.AdminHandler")),
	}
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/material")
	g.POST("/list", ginx.BS[ListMaterialsReq](h.List))
	g.POST("/accept", ginx.BS[AcceptMaterialReq](h.Accept))
	g.POST("/notify", ginx.BS[NotifyUserReq](h.Notify))
}

func (h *AdminHandler) List(ctx *ginx.Context, req ListMaterialsReq, _ session.Session) (ginx.Result, error) {
	materials, total, err := h.svc.List(ctx.Request.Context(), req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, fmt.Errorf("获取兑换码失败: %w", err)
	}
	return ginx.Result{
		Data: ListMaterialsResp{
			Total: total,
			Materials: slice.Map(materials, func(idx int, src domain.Material) Material {
				return Material{
					ID:        src.ID,
					Uid:       src.Uid,
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

func (h *AdminHandler) Accept(ctx *ginx.Context, req AcceptMaterialReq, _ session.Session) (ginx.Result, error) {
	err := h.svc.Accept(ctx.Request.Context(), req.ID)
	if err != nil {
		return systemErrorResult, fmt.Errorf("更新素材状态为接受状态失败:%w", err)
	}
	m, err := h.svc.FindByID(ctx.Request.Context(), req.ID)
	if err != nil {
		return systemErrorResult, fmt.Errorf("素材未找到：%w", err)
	}
	evt := event.MemberEvent{
		Key:    fmt.Sprintf("material-accepted-%d", time.Now().UnixMilli()),
		Uid:    m.Uid,
		Days:   30,
		Biz:    "material",
		BizId:  m.ID,
		Action: "素材被采纳",
	}
	if er := h.producer.Produce(ctx, evt); er != nil {
		h.logger.Error("为素材被接受的用户发送福利失败",
			elog.FieldErr(er),
			elog.Any("event", evt),
		)
	}
	return ginx.Result{Msg: "OK"}, nil
}

func (h *AdminHandler) Notify(ctx *ginx.Context, req NotifyUserReq, _ session.Session) (ginx.Result, error) {
	m, err := h.svc.FindByID(ctx.Request.Context(), req.ID)
	if err != nil {
		return systemErrorResult, fmt.Errorf("素材未找到：%w", err)
	}

	if !m.Status.IsAccepted() {
		return systemErrorResult, fmt.Errorf("素材未被接受到：%w", err)
	}
	// 根据素材中关联的uid查找手机号
	u, err := h.userSvc.Profile(ctx.Request.Context(), m.Uid)
	if err != nil {
		return systemErrorResult, fmt.Errorf("用户未找到：%w", err)
	}
	if u.Phone == "" {
		return phoneNotLinkedErrorResult, errors.New("用户未绑定手机号")
	}
	// 构建短信请求
	const templateID = "SMS_491540609"
	resp, err := h.smsClient.Send(sms.SendReq{
		PhoneNumbers: []string{u.Phone},
		TemplateID:   templateID,
		TemplateParam: map[string]string{
			"date": req.Date,
		},
	})
	if err != nil {
		return systemErrorResult, errors.New("发送通知短信失败")
	}
	r, ok := resp.PhoneNumbers[u.Phone]
	if !ok || r.Code != sms.OK {
		return notifyFailedErrorResult, errors.New("用户无法收到通知")
	}
	return ginx.Result{Msg: "OK"}, nil
}
