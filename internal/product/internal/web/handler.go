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
	"github.com/ecodeclub/webook/internal/product/internal/domain"
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
	g.POST("/spu/detail", ginx.BS[SNReq](h.RetrieveSPUDetail))
	g.POST("/sku/detail", ginx.BS[SNReq](h.RetrieveSKUDetail))
	g.POST("/save", ginx.BS[SPUSaveReq](h.SaveProduct))
	g.POST("/spu/list", ginx.BS[SPUListReq](h.SPUList))
}

func (h *Handler) RetrieveSPUDetail(ctx *ginx.Context, req SNReq, _ session.Session) (ginx.Result, error) {
	spu, err := h.svc.FindSPUBySN(ctx.Request.Context(), req.SN)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: SPU{
			SN:   spu.SN,
			Name: spu.Name,
			Desc: spu.Desc,
			SKUs: slice.Map(spu.SKUs, func(idx int, src domain.SKU) SKU {
				return h.toSKU(src)
			}),
		},
	}, nil
}

func (h *Handler) toSKU(sku domain.SKU) SKU {
	return SKU{
		SN:         sku.SN,
		Name:       sku.Name,
		Desc:       sku.Desc,
		Price:      sku.Price,
		Stock:      sku.Stock,
		StockLimit: sku.StockLimit,
		SaleType:   sku.SaleType.ToUint8(),
		Attrs:      sku.Attrs,
		Image:      sku.Image,
	}
}

func (h *Handler) RetrieveSKUDetail(ctx *ginx.Context, req SNReq, _ session.Session) (ginx.Result, error) {
	sku, err := h.svc.FindSKUBySN(ctx.Request.Context(), req.SN)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toSKU(sku),
	}, nil
}

func (h *Handler) SaveProduct(ctx *ginx.Context, req SPUSaveReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	id, err := h.svc.SaveProduct(ctx.Request.Context(), req.SPU.newDomainSPU(), uid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) SPUList(ctx *ginx.Context, req SPUListReq, _ session.Session) (ginx.Result, error) {
	count, products, err := h.svc.ProductList(ctx.Context, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: SPUListResp{
			List: slice.Map(products, func(idx int, src domain.SPU) SPU {
				return h.toSPU(src)
			}),
			Count: count,
		},
	}, nil

}

func (h *Handler) toSPU(sku domain.SPU) SPU {
	return SPU{
		ID:   sku.ID,
		SN:   sku.SN,
		Name: sku.Name,
		Desc: sku.Desc,
		Category1: Category{
			Name: sku.Category1,
		},
		Category0: Category{
			Name: sku.Category0,
		},
	}
}
