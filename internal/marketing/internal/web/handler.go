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
	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/service"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/code")
	g.POST("/redeem", ginx.BS[RedeemRedemptionCodeReq](h.RedeemRedemptionCode))
	g.POST("/list", ginx.BS[ListRedemptionCodesReq](h.ListRedemptionCodes))

	i := server.Group("/invitation")
	i.POST("/gen", ginx.S(h.GenerateInvitationCode))
}

func (h *Handler) PublicRoutes(_ *gin.Engine) {}

func (h *Handler) RedeemRedemptionCode(ctx *ginx.Context, req RedeemRedemptionCodeReq, sess session.Session) (ginx.Result, error) {
	err := h.svc.RedeemRedemptionCode(ctx.Request.Context(), sess.Claims().Uid, req.Code)
	if err != nil {
		if errors.Is(err, service.ErrRedemptionCodeUsed) {
			return redemptionCodeUsedErrResult, err
		}
		if errors.Is(err, service.ErrRedemptionNotFound) {
			return redemptionCodeNotFoundErrResult, err
		}
		return systemErrorResult, err
	}
	return ginx.Result{Msg: "OK"}, nil
}

func (h *Handler) ListRedemptionCodes(ctx *ginx.Context, req ListRedemptionCodesReq, sess session.Session) (ginx.Result, error) {
	codes, total, err := h.svc.ListRedemptionCodes(ctx.Request.Context(), sess.Claims().Uid, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, fmt.Errorf("获取个人兑换码失败: %w", err)
	}
	return ginx.Result{
		Data: ListRedemptionCodesResp{
			Total: total,
			Codes: slice.Map(codes, func(idx int, src domain.RedemptionCode) RedemptionCode {
				return RedemptionCode{
					Code: src.Code,
					Type: src.Type,
					SKU: SKU{
						SN:   src.Attrs.SKU.SN,
						Name: src.Attrs.SKU.Name,
					},
					Status: src.Status.ToUint8(),
					Utime:  src.Utime,
				}
			}),
		},
	}, nil
}

func (h *Handler) GenerateInvitationCode(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	c, err := h.svc.GenerateInvitationCode(ctx, sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, fmt.Errorf("生成邀请码失败: %w", err)
	}
	return ginx.Result{Data: c.Code}, nil
}
