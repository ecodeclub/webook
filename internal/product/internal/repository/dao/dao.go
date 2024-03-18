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
	"time"

	"github.com/ego-component/egorm"
	"github.com/shopspring/decimal"
)

type ProductDAO interface {
	FindSPUByID(ctx context.Context, id int64) (ProductSPU, error)
	FindSKUBySN(ctx context.Context, sn string) (ProductSKU, error)
	CreateSPU(ctx context.Context, spu ProductSPU) (int64, error)
	CreateSKU(ctx context.Context, sku ProductSKU) (int64, error)
}

type ProductGORMDAO struct {
	db *egorm.Component
}

func NewProductGORMDAO(db *egorm.Component) ProductDAO {
	return &ProductGORMDAO{db: db}
}

func (d *ProductGORMDAO) FindSKUBySN(ctx context.Context, sn string) (ProductSKU, error) {
	var res ProductSKU
	err := d.db.WithContext(ctx).Where("sn = ? AND status = ?", sn, StatusOnShelf).First(&res).Error
	return res, err
}

func (d *ProductGORMDAO) FindSPUByID(ctx context.Context, id int64) (ProductSPU, error) {
	var res ProductSPU
	err := d.db.WithContext(ctx).Where("id = ? AND status = ?", id, StatusOnShelf).First(&res).Error
	return res, err
}

func (d *ProductGORMDAO) CreateSPU(ctx context.Context, spu ProductSPU) (int64, error) {
	now := time.Now()
	spu.Utime, spu.Ctime = now.UnixMilli(), now.UnixMilli()
	return spu.Id, d.db.WithContext(ctx).Create(&spu).Error
}

func (d *ProductGORMDAO) CreateSKU(ctx context.Context, sku ProductSKU) (int64, error) {
	now := time.Now()
	sku.Utime, sku.Ctime = now.UnixMilli(), now.UnixMilli()
	return sku.Id, d.db.WithContext(ctx).Create(&sku).Error
}

type ProductSPU struct {
	Id          int64  `gorm:"primaryKey;autoIncrement;comment:SPU ID"`
	SN          string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_code;comment:SPU 对外展示ID"`
	Name        string `gorm:"type:varchar(255);not null;comment:商品名称"`
	Description string `gorm:"not null; comment:商品描述"`
	Status      int64  `gorm:"type:tinyint unsigned;not null;default:0;comment:状态 0=下架 1=上架"`
	Ctime       int64
	Utime       int64
}

type ProductSKU struct {
	Id           int64           `gorm:"primaryKey;autoIncrement;comment:SKU ID"`
	SN           string          `gorm:"type:varchar(255);not null;uniqueIndex:uniq_code;comment:SKU 对外展示ID"`
	ProductSPUID int64           `gorm:"column:product_spu_id;not null;index:idx_product_spu_id,comment:所属SPU的ID不是SN"`
	Name         string          `gorm:"type:varchar(255);not null;comment:SKU名称"`
	Description  string          `gorm:"not null;comment:SKU描述"`
	Price        decimal.Decimal `gorm:"type:decimal(10,2);not null;comment:价格"`
	Stock        int64           `gorm:"not null;default:0;comment:库存数量"`
	StockLimit   int64           `gorm:"not null;comment:库存限制"`
	SaleType     int64           `gorm:"type:tinyint unsigned;not null;default:1;comment:销售类型: 1=无限期 2=限时促销 3=预售"`
	// SaleStart    sql.NullInt64   `gorm:"comment:销售开始时间,无限期销售为NULL"`
	// SaleEnd      sql.NullInt64   `gorm:"comment:销售结束时间,无限期和预售为NULL"`
	Status int64 `gorm:"type:tinyint unsigned;not null;default:0;comment:状态 0=下架 1=上架"`
	Ctime  int64
	Utime  int64
}

const (
	StatusOffShelf = iota // 下架
	StatusOnShelf         // 上架
)
