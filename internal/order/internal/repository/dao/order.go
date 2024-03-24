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
	"gorm.io/gorm"
)

type OrderDAO interface {
	CreateOrder(ctx context.Context, o Order, items []OrderItem) (int64, error)
	UpdateOrder(ctx context.Context, order Order) error

	FindOrderBySN(ctx context.Context, sn string) (Order, error)
	FindOrderBySNAndBuyerID(ctx context.Context, sn string, buyerID int64) (Order, error)
	FindOrderItemsByOrderID(ctx context.Context, oid int64) ([]OrderItem, error)
	Count(ctx context.Context, uid int64) (int64, error)
	List(ctx context.Context, offset int, limit int, uid int64) ([]Order, error)
}

func NewOrderGORMDAO(db *egorm.Component) OrderDAO {
	return &gormOrderDAO{db: db}
}

type gormOrderDAO struct {
	db *egorm.Component
}

func (g *gormOrderDAO) CreateOrder(ctx context.Context, order Order, items []OrderItem) (int64, error) {
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		order.Ctime, order.Utime = now.UnixMilli(), now.UnixMilli()
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		for i := 0; i < len(items); i++ {
			items[i].OrderId = order.Id
			items[i].Ctime, items[i].Utime = now.UnixMilli(), now.UnixMilli()
		}
		if err := tx.Create(&items).Error; err != nil {
			return err
		}
		return nil
	})
	return order.Id, err
}

func (g *gormOrderDAO) UpdateOrder(ctx context.Context, order Order) error {
	order.Utime = time.Now().UnixMilli()
	return g.db.WithContext(ctx).Where("id = ?", order.Id).Updates(&order).Error
}

func (g *gormOrderDAO) FindOrderBySN(ctx context.Context, sn string) (Order, error) {
	var res Order
	err := g.db.WithContext(ctx).First(&res, "sn = ?", sn).Error
	return res, err
}

func (g *gormOrderDAO) FindOrderBySNAndBuyerID(ctx context.Context, sn string, buyerID int64) (Order, error) {
	var res Order
	err := g.db.WithContext(ctx).First(&res, "sn = ? AND buyer_id = ?", sn, buyerID).Error
	return res, err
}

func (g *gormOrderDAO) FindOrderItemsByOrderID(ctx context.Context, oid int64) ([]OrderItem, error) {
	var res []OrderItem
	err := g.db.WithContext(ctx).Find(&res, "order_id = ?", oid).Error
	return res, err
}

func (g *gormOrderDAO) Count(ctx context.Context, uid int64) (int64, error) {
	var res int64
	db := g.db.WithContext(ctx).Model(&Order{})
	if uid != 0 {
		db = db.Where("uid = ?", uid)
	}
	err := db.Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (g *gormOrderDAO) List(ctx context.Context, offset int, limit int, uid int64) ([]Order, error) {
	var res []Order
	db := g.db.WithContext(ctx)
	if uid != 0 {
		db = db.Where("uid = ?", uid)
	}
	err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&res).Error
	return res, err
}

type Order struct {
	Id                 int64  `gorm:"primaryKey;autoIncrement;comment:订单自增ID"`
	SN                 string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_order_sn;comment:订单序列号"`
	BuyerId            int64  `gorm:"not null;index:idx_buyer_id,comment:购买者ID"`
	PaymentId          int64  `gorm:"uniqueIndex:uniq_payment_id,comment:支付自增ID,冗余允许为NULL"`
	PaymentSn          string `gorm:"type:varchar(255);uniqueIndex:uniq_payment_sn;comment:支付序列号,冗余允许为NULL"`
	OriginalTotalPrice int64  `gorm:"not null;comment:原始总价;单位为分, 999表示9.99元"`
	RealTotalPrice     int64  `gorm:"not null;comment:实付总价;单位为分, 999表示9.99元"`
	ClosedAt           int64  `gorm:"comment:订单关闭时间"`
	Status             int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:订单状态 1=未支付 2=已完成(用户支付完成) 3=已关闭(用户主动取消) 4=已超时(订单超时关闭)"`
	Ctime              int64
	Utime              int64
}

type OrderItem struct {
	Id               int64  `gorm:"primaryKey;autoIncrement;comment:订单项自增ID"`
	OrderId          int64  `gorm:"not null;index:idx_order_id,comment:订单自增ID"`
	SPUId            int64  `gorm:"column:spu_id;not null;comment:SPU自增ID"`
	SKUId            int64  `gorm:"column:sku_id;not null;index:idx_sku_id,comment:SKU自增ID"`
	SKUName          string `gorm:"column:sku_name;type:varchar(255);not null;comment:SKU名称"`
	SKUDescription   string `gorm:"column:sku_description;not null;comment:SKU描述"`
	SKUOriginalPrice int64  `gorm:"column:sku_original_price;not null;comment:商品原始单价;单位为分, 999表示9.99元"`
	SKURealPrice     int64  `gorm:"column:sku_real_price;not null;comment:商品实付单价;单位为分, 999表示9.99元"`
	Quantity         int64  `gorm:"not null;comment:购买数量"`
	Ctime            int64
	Utime            int64
}
