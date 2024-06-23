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

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/lithammer/shortuuid/v4"

	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/product/internal/domain"
	"github.com/ecodeclub/webook/internal/product/internal/repository/dao"
	"github.com/gotomicro/ego/core/elog"
)

type ProductRepository interface {
	FindSPUBySN(ctx context.Context, sn string) (domain.SPU, error)
	FindSPUByID(ctx context.Context, id int64) (domain.SPU, error)
	FindSKUBySN(ctx context.Context, sn string) (domain.SKU, error)
	SaveSPU(ctx context.Context, spu domain.SPU) (string, error)
	FindSPUs(ctx context.Context, offset, limit int) (int64, []domain.SPU, error)
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
		ID:        spu.Id,
		SN:        spu.SN,
		Name:      spu.Name,
		Desc:      spu.Description,
		Category0: spu.Category0,
		Category1: spu.Category1,
		Status:    domain.Status(spu.Status),
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

func (p *productRepository) SaveSPU(ctx context.Context, spu domain.SPU) (string, error) {
	spuEntity, skuEntities := p.toEntity(spu)
	return spuEntity.SN, p.dao.SaveProduct(ctx, spuEntity, skuEntities)
}
func (p *productRepository) FindSPUs(ctx context.Context, offset, limit int) (int64, []domain.SPU, error) {
	var eg errgroup.Group
	var count int64
	var spus []dao.SPU
	eg.Go(func() error {
		var err error
		spus, err = p.dao.FindSPUs(ctx, offset, limit)
		return err
	})
	eg.Go(func() error {
		var err error
		count, err = p.dao.CountSPUs(ctx)
		return err
	})
	if err := eg.Wait(); err != nil {
		return 0, nil, err
	}
	domainSPUs := make([]domain.SPU, 0, len(spus))
	for _, spu := range spus {
		domainSPUs = append(domainSPUs, p.toDomainSPU(spu, []dao.SKU{}))
	}
	return count, domainSPUs, nil
}

func (p *productRepository) toEntity(spu domain.SPU) (dao.SPU, []dao.SKU) {
	spuEntity := dao.SPU{
		Id:          spu.ID,
		Category0:   spu.Category0,
		Category1:   spu.Category1,
		SN:          spu.SN,
		Name:        spu.Name,
		Description: spu.Desc,
		Status:      spu.Status.ToUint8(),
	}
	if spu.SN == "" {
		spuEntity.SN = p.genSN()
	}
	skus := make([]dao.SKU, 0, len(spu.SKUs))
	for _, domainSku := range spu.SKUs {
		sku := p.toSKUEntity(domainSku)
		if domainSku.SN == "" {
			sku.SN = p.genSN()
		}
		skus = append(skus, sku)
	}
	return spuEntity, skus
}

func (p *productRepository) toSKUEntity(sku domain.SKU) dao.SKU {
	skuEntity := dao.SKU{
		SPUID:       sku.SPUID,
		Id:          sku.ID,
		SN:          sku.SN,
		Name:        sku.Name,
		Description: sku.Desc,
		Price:       sku.Price,
		Stock:       sku.Stock,
		StockLimit:  sku.StockLimit,
		SaleType:    sku.SaleType.ToUint8(),
		Image:       sku.Image,
		Status:      sku.Status.ToUint8(),
		Attrs:       sqlx.NewNullString(sku.Attrs),
	}
	return skuEntity
}

func (p *productRepository) genSN() string {
	return shortuuid.New()
}
