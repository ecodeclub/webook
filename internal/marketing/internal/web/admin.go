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
	"context"
	"fmt"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/marketing/internal/domain"
	"github.com/ecodeclub/webook/internal/marketing/internal/service"
	"github.com/ecodeclub/webook/internal/product"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	svc                     service.RedemptionCodeAdminService
	productSvc              product.Service
	redemptionCodeGenerator func(id int64) string
}

func NewAdminHandler(svc service.RedemptionCodeAdminService, productSvc product.Service, redemptionCodeGenerator func(id int64) string) *AdminHandler {
	return &AdminHandler{
		svc:                     svc,
		productSvc:              productSvc,
		redemptionCodeGenerator: redemptionCodeGenerator,
	}
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/code")
	g.POST("/gen", ginx.BS[GenerateRedemptionCodeReq](h.GenerateRedemptionCode))
}

func (h *AdminHandler) GenerateRedemptionCode(ctx *ginx.Context, req GenerateRedemptionCodeReq, sess session.Session) (ginx.Result, error) {
	codes, err := h.generateCodes(ctx.Request.Context(), req)
	if err != nil {
		return systemErrorResult, err
	}
	err = h.svc.GenerateRedemptionCodes(ctx.Request.Context(), codes)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{Msg: "OK"}, nil
}

func (h *AdminHandler) generateCodes(ctx context.Context, req GenerateRedemptionCodeReq) ([]domain.RedemptionCode, error) {
	sku, err := h.productSvc.FindSKUBySN(ctx, req.SKUSN)
	if err != nil {
		return nil, fmt.Errorf("获取SKU信息失败: %w", err)
	}
	spu, err := h.productSvc.FindSPUByID(ctx, sku.SPUID)
	if err != nil {
		return nil, fmt.Errorf("获取SPU信息失败: %w", err)
	}
	codes := make([]domain.RedemptionCode, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		codes = append(codes, h.generateCode(req, spu, sku))
	}
	return codes, nil
}

func (h *AdminHandler) generateCode(req GenerateRedemptionCodeReq, spu product.SPU, sku product.SKU) domain.RedemptionCode {
	return domain.RedemptionCode{
		OwnerID: 0, // ownerID为0表示admin
		Biz:     req.Biz,
		BizId:   req.BizId,
		Type:    spu.Category1,
		Attrs: domain.CodeAttrs{SKU: domain.SKU{
			ID:    sku.ID,
			SN:    sku.SN,
			Name:  sku.Name,
			Attrs: sku.Attrs,
		}},
		Code:   h.redemptionCodeGenerator(req.BizId),
		Status: domain.RedemptionCodeStatusUnused,
	}
}
