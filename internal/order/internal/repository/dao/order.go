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

	"github.com/ecodeclub/ekit/sqlx"
	"github.com/ecodeclub/webook/internal/order/internal/domain"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

type OrderDAO interface {
	CreateOrder(ctx context.Context, o Order, items []OrderItem) (int64, error)
	UpdateUnpaidOrderPaymentInfo(ctx context.Context, uid, oid, pid int64, psn string) error
	FindOrderByUIDAndSNAndStatus(ctx context.Context, uid int64, sn string, status uint8) (Order, error)
	FindOrderItemsByOrderID(ctx context.Context, oid int64) ([]OrderItem, error)
	CountOrdersByUID(ctx context.Context, uid int64, status uint8) (int64, error)
	FindOrdersByUID(ctx context.Context, offset, limit int, uid int64, status uint8) ([]Order, error)
	SetOrderCanceled(ctx context.Context, uid, oid int64) error
	SetOrderStatus(ctx context.Context, uid int64, orderSN string, status uint8) error
	FindTimeoutOrders(ctx context.Context, offset, limit int, ctime int64) ([]Order, error)
	CountTimeoutOrders(ctx context.Context, ctime int64) (int64, error)
	SetOrdersTimeoutClosed(ctx context.Context, orderIDs []int64, ctime int64) error
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

func (g *gormOrderDAO) UpdateUnpaidOrderPaymentInfo(ctx context.Context, uid, oid, pid int64, psn string) error {
	order := Order{PaymentId: sqlx.NewNullInt64(pid), PaymentSn: sqlx.NewNullString(psn), Status: domain.StatusProcessing.ToUint8(), Utime: time.Now().UnixMilli()}
	return g.db.WithContext(ctx).
		Where("buyer_id = ? AND id = ? AND status = ?", uid, oid, domain.StatusInit.ToUint8()).
		Updates(order).Error
}

func (g *gormOrderDAO) FindOrderByUIDAndSNAndStatus(ctx context.Context, uid int64, sn string, status uint8) (Order, error) {
	var res Order
	err := g.db.WithContext(ctx).
		Where("buyer_id = ? AND sn = ? AND status >= ?", uid, sn, status).
		First(&res).Error
	return res, err
}

func (g *gormOrderDAO) FindOrderItemsByOrderID(ctx context.Context, oid int64) ([]OrderItem, error) {
	var res []OrderItem
	err := g.db.WithContext(ctx).Order("ctime DESC").Find(&res, "order_id = ?", oid).Error
	return res, err
}

func (g *gormOrderDAO) CountOrdersByUID(ctx context.Context, uid int64, status uint8) (int64, error) {
	var res int64
	query := g.db.WithContext(ctx).Model(&Order{})
	if uid > 0 {
		query = query.Where("buyer_id = ?", uid)
	}
	if status > 0 {
		query = query.Where("status >= ?", status)
	}
	err := query.Count(&res).Error
	return res, err
}

func (g *gormOrderDAO) FindOrdersByUID(ctx context.Context, offset, limit int, uid int64, status uint8) ([]Order, error) {
	var res []Order
	query := g.db.WithContext(ctx).Model(&Order{}).Offset(offset).Limit(limit).Order("ctime DESC")
	if uid > 0 {
		query = query.Where("buyer_id = ?", uid)
	}
	if status > 0 {
		query = query.Where("status >= ?", status)
	}
	err := query.Find(&res).Error
	return res, err
}

func (g *gormOrderDAO) SetOrderCanceled(ctx context.Context, uid, oid int64) error {
	order := Order{Status: domain.StatusCanceled.ToUint8(), Utime: time.Now().UnixMilli()}
	return g.db.WithContext(ctx).Where("buyer_id = ? AND id = ? AND status = ?", uid, oid, domain.StatusProcessing.ToUint8()).Updates(order).Error
}

func (g *gormOrderDAO) SetOrderStatus(ctx context.Context, uid int64, orderSN string, status uint8) error {
	order := Order{Status: status, Utime: time.Now().UnixMilli()}
	return g.db.WithContext(ctx).Where("buyer_id = ? AND sn = ?", uid, orderSN).Updates(order).Error
}

func (g *gormOrderDAO) FindTimeoutOrders(ctx context.Context, offset, limit int, ctime int64) ([]Order, error) {
	var res []Order
	err := g.db.WithContext(ctx).Offset(offset).Limit(limit).Order("ctime DESC").
		Where("status <= ? AND Ctime <= ?", domain.StatusProcessing.ToUint8(), ctime).
		Find(&res).Error
	return res, err
}

func (g *gormOrderDAO) CountTimeoutOrders(ctx context.Context, ctime int64) (int64, error) {
	var res int64
	err := g.db.WithContext(ctx).Model(&Order{}).
		Where("status <= ? AND Ctime <= ?", domain.StatusProcessing.ToUint8(), ctime).
		Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (g *gormOrderDAO) SetOrdersTimeoutClosed(ctx context.Context, orderIDs []int64, ctime int64) error {
	timestamp := time.Now().UnixMilli()
	return g.db.WithContext(ctx).Model(&Order{}).
		Where("status <= ? AND Ctime <= ? AND id IN ?", domain.StatusProcessing.ToUint8(), ctime, orderIDs).
		Updates(map[string]any{
			"status": domain.StatusTimeoutClosed.ToUint8(),
			"utime":  timestamp,
		}).Error
}

type Order struct {
	Id               int64          `gorm:"primaryKey;autoIncrement;comment:订单自增ID"`
	SN               string         `gorm:"type:varchar(255);not null;uniqueIndex:uniq_order_sn;comment:订单序列号"`
	BuyerId          int64          `gorm:"not null;index:idx_buyer_id;comment:购买者ID"`
	PaymentId        sql.NullInt64  `gorm:"uniqueIndex:uniq_payment_id;comment:支付自增ID,冗余允许为NULL"`
	PaymentSn        sql.NullString `gorm:"type:varchar(255);uniqueIndex:uniq_payment_sn;comment:支付序列号,冗余允许为NULL"`
	OriginalTotalAmt int64          `gorm:"not null;comment:原始总价;单位为分, 999表示9.99元"`
	RealTotalAmt     int64          `gorm:"not null;comment:实付总价;单位为分, 999表示9.99元"`
	Status           uint8          `gorm:"type:tinyint unsigned;not null;default:1;index:idx_order_status;comment:订单状态 1=未支付 2=处理中 3=支付成功(用户支付完成) 4=支付失败 5=已取消(用户主动取消) 6=已过期(订单超时关闭)"`
	Ctime            int64
	Utime            int64
}

type OrderItem struct {
	Id               int64  `gorm:"primaryKey;autoIncrement;comment:订单项自增ID"`
	OrderId          int64  `gorm:"not null;index:idx_order_id;comment:订单自增ID"`
	SPUId            int64  `gorm:"column:spu_id;not null;index:idx_spu_id;comment:SPU自增ID"`
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
