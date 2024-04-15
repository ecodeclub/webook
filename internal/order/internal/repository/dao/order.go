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

	"github.com/ecodeclub/webook/internal/order/internal/domain"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

type OrderDAO interface {
	CreateOrder(ctx context.Context, o Order, items []OrderItem) (int64, error)
	UpdateOrderPaymentIDAndPaymentSN(ctx context.Context, uid, oid, pid int64, psn string) error
	FindOrderByUIDAndSN(ctx context.Context, uid int64, sn string) (Order, error)
	FindOrderItemsByOrderID(ctx context.Context, oid int64) ([]OrderItem, error)
	CountOrdersByUID(ctx context.Context, uid int64) (int64, error)
	FindOrdersByUID(ctx context.Context, uid int64, offset, limit int) ([]Order, error)
	CancelOrder(ctx context.Context, uid, oid int64) error
	CompleteOrder(ctx context.Context, uid, oid int64) error

	FindExpiredOrders(ctx context.Context, offset, limit int, ctime int64) ([]Order, error)
	CountExpiredOrders(ctx context.Context, ctime int64) (int64, error)
	CloseExpiredOrders(ctx context.Context, orderIDs []int64, ctime int64) error

	FindOrderBySN(ctx context.Context, sn string) (Order, error)
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

func (g *gormOrderDAO) UpdateOrderPaymentIDAndPaymentSN(ctx context.Context, uid, oid, pid int64, psn string) error {
	order := Order{PaymentId: pid, PaymentSn: psn, Utime: time.Now().UnixMilli()}
	return g.db.WithContext(ctx).Where("buyer_id = ? AND id = ? AND status = ?", uid, oid, domain.StatusUnpaid.ToUint8()).Updates(order).Error
}

func (g *gormOrderDAO) FindOrderByUIDAndSN(ctx context.Context, uid int64, sn string) (Order, error) {
	var res Order
	err := g.db.WithContext(ctx).First(&res, "buyer_id = ? AND sn = ?", uid, sn).Error
	return res, err
}

func (g *gormOrderDAO) FindOrderItemsByOrderID(ctx context.Context, oid int64) ([]OrderItem, error) {
	var res []OrderItem
	err := g.db.WithContext(ctx).Find(&res, "order_id = ?", oid).Error
	return res, err
}

func (g *gormOrderDAO) CountOrdersByUID(ctx context.Context, uid int64) (int64, error) {
	var res int64
	err := g.db.WithContext(ctx).Model(&Order{}).Where("buyer_id = ?", uid).Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (g *gormOrderDAO) FindOrdersByUID(ctx context.Context, uid int64, offset, limit int) ([]Order, error) {
	var res []Order
	err := g.db.WithContext(ctx).Offset(offset).Limit(limit).Order("id DESC").Find(&res, "buyer_id = ?", uid).Error
	return res, err
}

func (g *gormOrderDAO) CancelOrder(ctx context.Context, uid, oid int64) error {
	order := Order{Status: domain.StatusCanceled.ToUint8(), Utime: time.Now().UnixMilli()}
	return g.db.WithContext(ctx).Where("buyer_id = ? AND id = ? AND status = ?", uid, oid, domain.StatusUnpaid.ToUint8()).Updates(order).Error
}

func (g *gormOrderDAO) CompleteOrder(ctx context.Context, uid, oid int64) error {
	// 已收到用户的付款,不管当前处于什么状态一律标记为已完成
	order := Order{Status: domain.StatusCompleted.ToUint8(), Utime: time.Now().UnixMilli()}
	return g.db.WithContext(ctx).Where("buyer_id = ? AND id = ?", uid, oid).Updates(order).Error
}

func (g *gormOrderDAO) FindExpiredOrders(ctx context.Context, offset, limit int, ctime int64) ([]Order, error) {
	var res []Order
	err := g.db.WithContext(ctx).Offset(offset).Limit(limit).Order("id DESC").
		Find(&res, "status = ? AND Ctime <= ?", domain.StatusUnpaid.ToUint8(), ctime).Error
	return res, err
}

func (g *gormOrderDAO) CountExpiredOrders(ctx context.Context, ctime int64) (int64, error) {
	var res int64
	err := g.db.WithContext(ctx).Model(&Order{}).
		Where("status = ? AND Ctime <= ?", domain.StatusUnpaid.ToUint8(), ctime).
		Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (g *gormOrderDAO) CloseExpiredOrders(ctx context.Context, orderIDs []int64, ctime int64) error {
	timestamp := time.Now().UnixMilli()
	return g.db.WithContext(ctx).Model(&Order{}).
		Where("status = ? AND Ctime <= ? AND id IN ?", domain.StatusUnpaid.ToUint8(), ctime, orderIDs).
		Updates(map[string]any{
			"status": domain.StatusExpired.ToUint8(),
			"utime":  timestamp,
		}).Error
}

func (g *gormOrderDAO) FindOrderBySN(ctx context.Context, sn string) (Order, error) {
	var res Order
	err := g.db.WithContext(ctx).First(&res, "sn = ?", sn).Error
	return res, err
}

type Order struct {
	Id                 int64  `gorm:"primaryKey;autoIncrement;comment:订单自增ID"`
	SN                 string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_order_sn;comment:订单序列号"`
	BuyerId            int64  `gorm:"not null;index:idx_buyer_id;comment:购买者ID"`
	PaymentId          int64  `gorm:"uniqueIndex:uniq_payment_id;comment:支付自增ID,冗余允许为NULL"`
	PaymentSn          string `gorm:"type:varchar(255);uniqueIndex:uniq_payment_sn;comment:支付序列号,冗余允许为NULL"`
	OriginalTotalPrice int64  `gorm:"not null;comment:原始总价;单位为分, 999表示9.99元"`
	RealTotalPrice     int64  `gorm:"not null;comment:实付总价;单位为分, 999表示9.99元"`
	Status             uint8  `gorm:"type:tinyint unsigned;not null;default:1;comment:订单状态 1=未支付 2=已完成(用户支付完成) 3=已关闭(用户主动取消) 4=已超时(订单超时关闭)"`
	Ctime              int64
	Utime              int64
}

type OrderItem struct {
	Id               int64  `gorm:"primaryKey;autoIncrement;comment:订单项自增ID"`
	OrderId          int64  `gorm:"not null;index:idx_order_id;comment:订单自增ID"`
	SPUId            int64  `gorm:"column:spu_id;not null;index:idx_spu_id;comment:SPU自增ID"`
	SPUSN            string `gorm:"column:spu_sn;type:varchar(255);not null;comment:SPU序列号"`
	SKUId            int64  `gorm:"column:sku_id;not null;index:idx_sku_id;comment:SKU自增ID"`
	SKUSN            string `gorm:"column:sku_sn;type:varchar(255);not null;comment:SKU序列号"`
	SKUImage         string `gorm:"type:varchar(512);not null;comment:SKU缩略图,CDN绝对路径"`
	SKUName          string `gorm:"column:sku_name;type:varchar(255);not null;comment:SKU名称"`
	SKUDescription   string `gorm:"column:sku_description;not null;comment:SKU描述"`
	SKUOriginalPrice int64  `gorm:"column:sku_original_price;not null;comment:商品原始单价;单位为分, 999表示9.99元"`
	SKURealPrice     int64  `gorm:"column:sku_real_price;not null;comment:商品实付单价;单位为分, 999表示9.99元"`
	Quantity         int64  `gorm:"not null;comment:购买数量"`
	Ctime            int64
	Utime            int64
}
