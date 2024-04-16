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

package dao

import (
	"context"
	"database/sql"
	"time"

	"github.com/ecodeclub/webook/internal/product/internal/domain"
	"github.com/ego-component/egorm"
)

type ProductDAO interface {
	FindSPUByID(ctx context.Context, id int64) (SPU, error)
	FindSPUBySN(ctx context.Context, sn string) (SPU, error)
	FindSKUBySN(ctx context.Context, sn string) (SKU, error)
	FindSKUsBySPUID(ctx context.Context, spuId int64) ([]SKU, error)
	CreateSPU(ctx context.Context, spu SPU) (int64, error)
	CreateSKU(ctx context.Context, sku SKU) (int64, error)
}

type ProductGORMDAO struct {
	db *egorm.Component
}

func NewProductGORMDAO(db *egorm.Component) ProductDAO {
	return &ProductGORMDAO{db: db}
}

func (d *ProductGORMDAO) FindSPUByID(ctx context.Context, id int64) (SPU, error) {
	var res SPU
	err := d.db.WithContext(ctx).Where("id = ? AND status = ?", id, domain.StatusOnShelf.ToUint8()).First(&res).Error
	return res, err
}

func (d *ProductGORMDAO) FindSPUBySN(ctx context.Context, sn string) (SPU, error) {
	var res SPU
	err := d.db.WithContext(ctx).Where("sn = ? AND status = ?", sn, domain.StatusOnShelf.ToUint8()).First(&res).Error
	return res, err
}

func (d *ProductGORMDAO) FindSKUBySN(ctx context.Context, sn string) (SKU, error) {
	var res SKU
	err := d.db.WithContext(ctx).Where("sn = ? AND status = ?", sn, domain.StatusOnShelf.ToUint8()).First(&res).Error
	return res, err
}

func (d *ProductGORMDAO) FindSKUsBySPUID(ctx context.Context, spuId int64) ([]SKU, error) {
	var res []SKU
	err := d.db.WithContext(ctx).Where("spu_id = ? AND status = ?", spuId, domain.StatusOnShelf.ToUint8()).
		Order("ctime DESC").
		Find(&res).Error
	return res, err
}

func (d *ProductGORMDAO) CreateSPU(ctx context.Context, spu SPU) (int64, error) {
	now := time.Now()
	spu.Utime, spu.Ctime = now.UnixMilli(), now.UnixMilli()
	return spu.Id, d.db.WithContext(ctx).Create(&spu).Error
}

func (d *ProductGORMDAO) CreateSKU(ctx context.Context, sku SKU) (int64, error) {
	now := time.Now()
	sku.Utime, sku.Ctime = now.UnixMilli(), now.UnixMilli()
	return sku.Id, d.db.WithContext(ctx).Create(&sku).Error
}

type SPU struct {
	Id          int64  `gorm:"primaryKey;autoIncrement;comment:商品SPU自增ID"`
	SN          string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_product_spu_sn;comment:商品SPU序列号"`
	Name        string `gorm:"type:varchar(255);not null;comment:商品名称"`
	Description string `gorm:"not null; comment:商品描述"`
	Status      uint8  `gorm:"type:tinyint unsigned;not null;default:1;comment:状态 1=下架 2=上架"`
	Ctime       int64
	Utime       int64
}

type SKU struct {
	Id          int64  `gorm:"primaryKey;autoIncrement;comment:商品SKU自增ID"`
	SN          string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_product_sku_sn;comment:商品SKU序列号"`
	SPUID       int64  `gorm:"column:spu_id;not null;index:idx_spu_id;comment:商品SPU自增ID"`
	Name        string `gorm:"type:varchar(255);not null;comment:SKU名称"`
	Description string `gorm:"not null;comment:商品描述"`
	Price       int64  `gorm:"not null;comment:商品单价;单位为分, 999表示9.99元"`
	Stock       int64  `gorm:"not null;comment:库存数量"`
	StockLimit  int64  `gorm:"not null;comment:库存限制"`
	SaleType    uint8  `gorm:"type:tinyint unsigned;not null;default:1;comment:销售类型: 1=无限期 2=限时促销 3=预售"`
	// SaleStart    sql.NullInt64   `gorm:"comment:销售开始时间,无限期销售为NULL"`
	// SaleEnd      sql.NullInt64   `gorm:"comment:销售结束时间,无限期和预售为NULL"`
	Attrs  sql.NullString `gorm:"comment:商品销售属性,JSON格式"`
	Image  string         `gorm:"type:varchar(512);not null;comment:商品缩略图,CDN绝对路径"`
	Status uint8          `gorm:"type:tinyint unsigned;not null;default:1;comment:状态 1=下架 2=上架"`
	Ctime  int64
	Utime  int64
}
