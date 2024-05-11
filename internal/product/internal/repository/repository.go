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

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/product/internal/domain"
	"github.com/ecodeclub/webook/internal/product/internal/repository/dao"
	"github.com/gotomicro/ego/core/elog"
)

type ProductRepository interface {
	FindSPUBySN(ctx context.Context, sn string) (domain.SPU, error)
	FindSPUByID(ctx context.Context, id int64) (domain.SPU, error)
	FindSKUBySN(ctx context.Context, sn string) (domain.SKU, error)
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

func (p *productRepository) FindSPUBySN(ctx context.Context, sn string) (domain.SPU, error) {
	spu, err := p.dao.FindSPUBySN(ctx, sn)
	if err != nil {
		return domain.SPU{}, err
	}
	skus, err := p.dao.FindSKUsBySPUID(ctx, spu.Id)
	if err != nil {
		return domain.SPU{}, err
	}
	return p.toDomainSPU(spu, skus), err
}

func (p *productRepository) toDomainSPU(spu dao.SPU, skus []dao.SKU) domain.SPU {
	return domain.SPU{
		ID:       spu.Id,
		SN:       spu.SN,
		Name:     spu.Name,
		Desc:     spu.Description,
		Category: spu.Category,
		Type:     spu.Type,
		Status:   domain.Status(spu.Status),
		SKUs: slice.Map(skus, func(idx int, src dao.SKU) domain.SKU {
			return p.toDomainSKU(src)
		}),
	}
}

func (p *productRepository) toDomainSKU(sku dao.SKU) domain.SKU {
	return domain.SKU{
		ID:         sku.Id,
		SPUID:      sku.SPUID,
		SN:         sku.SN,
		Name:       sku.Name,
		Desc:       sku.Description,
		Price:      sku.Price,
		Stock:      sku.Stock,
		StockLimit: sku.StockLimit,
		SaleType:   domain.SaleType(sku.SaleType),
		Attrs:      sku.Attrs.String,
		Image:      sku.Image,
		Status:     domain.Status(sku.Status),
	}
}

func (p *productRepository) FindSPUByID(ctx context.Context, id int64) (domain.SPU, error) {
	spu, err := p.dao.FindSPUByID(ctx, id)
	if err != nil {
		return domain.SPU{}, err
	}
	skus, err := p.dao.FindSKUsBySPUID(ctx, spu.Id)
	if err != nil {
		return domain.SPU{}, err
	}
	return p.toDomainSPU(spu, skus), err
}

func (p *productRepository) FindSKUBySN(ctx context.Context, sn string) (domain.SKU, error) {
	sku, err := p.dao.FindSKUBySN(ctx, sn)
	if err != nil {
		return domain.SKU{}, err
	}
	return p.toDomainSKU(sku), err
}
