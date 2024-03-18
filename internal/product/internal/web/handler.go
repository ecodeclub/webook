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
	"github.com/ecodeclub/webook/internal/product/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc service.Service
}

func NewHandler(svc service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/product")
	g.POST("/detail", ginx.BS[ProductSNReq](h.RetrieveProductDetail))
}

func (h *Handler) RetrieveProductDetail(ctx *ginx.Context, req ProductSNReq, _ session.Session) (ginx.Result, error) {
	p, err := h.svc.FindBySN(ctx.Request.Context(), req.SN)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: Product{
			SPU: ProductSPU{
				SN:   p.SPU.SN,
				Name: p.SPU.Name,
				Desc: p.SPU.Desc,
			},
			SKU: ProductSKU{
				SN:         p.SKU.SN,
				Name:       p.SKU.Name,
				Desc:       p.SKU.Desc,
				Price:      p.SKU.Price.String(),
				Stock:      p.SKU.Stock,
				StockLimit: p.SKU.StockLimit,
				SaleType:   p.SKU.SaleType,
			},
		},
	}, nil
}
