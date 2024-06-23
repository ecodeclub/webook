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

import "github.com/ecodeclub/webook/internal/product/internal/domain"

type SNReq struct {
	SN string `json:"sn"`
}
type SPUSaveReq struct {
	SPU SPU `json:"spu"`
}

type SPUSaveResp struct {
	ID int64 `json:"id"`
}

type SPUListReq struct {
	Offset int `json:"offset"`
	Limit int `json:"limit"`
}

type SPUListResp struct {
	List []SPU `json:"list"`
	Count int64`json:"count"`
}


type SPU struct {
	ID        int64    `json:"id,omitempty"`
	SN        string   `json:"sn"`
	Name      string   `json:"name"`
	Desc      string   `json:"desc"`
	SKUs      []SKU    `json:"skus,omitempty"`
	Category0 Category `json:"category0,omitempty"`
	Category1 Category `json:"category1,omitempty"`
}

type SKU struct {
	ID         int64  `json:"id,omitempty"`
	SN         string `json:"sn"`
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	Price      int64  `json:"price"`
	Stock      int64  `json:"stock"`
	StockLimit int64  `json:"stockLimit"`
	SaleType   uint8  `json:"saleType"`
	Attrs      string `json:"attrs,omitempty"`
	Image      string `json:"image"`
	// SaleStart  int64  `json:"saleStart"`
	// SaleEnd    int64  `json:"saleEnd"`
}

type Category struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

func (s SPU) newDomainSPU() domain.SPU {
	domainSPU := domain.SPU{
		ID:        s.ID,
		SN:        s.SN,
		Name:      s.Name,
		Desc:      s.Desc,
		Category0: s.Category0.Name,
		Category1: s.Category1.Name,
	}
	skus := make([]domain.SKU, 0, len(s.SKUs))
	for _, sku := range s.SKUs {
		skus = append(skus, sku.newDomainSKU())
	}
	domainSPU.SKUs = skus
	return domainSPU
}

func (s SKU) newDomainSKU() domain.SKU {
	return domain.SKU{
		ID:         s.ID,
		SN:         s.SN,
		Name:       s.Name,
		Desc:       s.Desc,
		Price:      s.Price,
		Stock:      s.Stock,
		StockLimit: s.StockLimit,
		SaleType:   domain.SaleType(s.SaleType),
		Attrs:      s.Attrs,
		Image:      s.Image,
	}
}