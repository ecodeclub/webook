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

package repository

import (
	"context"

	"github.com/ecodeclub/webook/internal/product/internal/domain"
	"github.com/ecodeclub/webook/internal/product/internal/repository/dao"
	"github.com/gotomicro/ego/core/elog"
)

type ProductRepository interface {
	FindBySN(ctx context.Context, sn string) (domain.Product, error)
}

func NewProductRepository(d dao.ProductDAO) ProductRepository {
	return &productRepository{
		dao:    d,
		logger: elog.DefaultLogger}
}

type productRepository struct {
	dao    dao.ProductDAO
	logger *elog.Component
}

func (p *productRepository) FindBySN(ctx context.Context, sn string) (domain.Product, error) {
	sku, err := p.dao.FindSKUBySN(ctx, sn)
	if err != nil {
		return domain.Product{}, err
	}
	spu, err := p.dao.FindSPUByID(ctx, sku.ProductSPUID)
	if err != nil {
		return domain.Product{}, err
	}
	return p.toDomainProduct(spu, sku), err
}

func (p *productRepository) toDomainProduct(spu dao.ProductSPU, sku dao.ProductSKU) domain.Product {
	return domain.Product{
		SPU: domain.SPU{
			ID:     spu.Id,
			SN:     spu.SN,
			Name:   spu.Name,
			Desc:   spu.Description,
			Status: spu.Status,
		},
		SKU: domain.SKU{
			ID:         sku.Id,
			SN:         sku.SN,
			Name:       sku.Name,
			Desc:       sku.Description,
			Price:      sku.Price,
			Stock:      sku.Stock,
			StockLimit: sku.StockLimit,
			SaleType:   sku.SaleType,
			Status:     sku.Status,
		},
	}
}
