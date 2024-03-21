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

import "context"

type OrderDAO interface {
	Create(ctx context.Context, o Order) (int64, error)
}

type Order struct {
	Id                 int64  `gorm:"primaryKey;autoIncrement;comment:订单自增ID"`
	SN                 string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_order_sn;comment:订单序列号"`
	BuyerId            int64  `gorm:"not null;index:idx_buyer_id,comment:购买者ID"`
	PaymentId          int64  `gorm:"not null;uniqueIndex:uniq_payment_id,comment:支付自增ID"`
	PaymentSN          string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_payment_sn;comment:支付序列号"`
	OriginalTotalPrice int64  `gorm:"not null;comment:原始总价;单位为分, 999表示9.99元"`
	RealTotalPrice     int64  `gorm:"not null;comment:实付总价;单位为分, 999表示9.99元"`
	ClosedAt           int64  `gorm:"comment:订单关闭时间"`
	Status             int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:订单状态 1=未支付 2=已完成 3=已关闭 4=已超时"`
	Ctime              int64
	Utime              int64
}

type OrderItem struct {
	Id               int64  `gorm:"primaryKey;autoIncrement;comment:订单项自增ID"`
	OrderId          int64  `gorm:"not null;index:idx_order_id,comment:订单自增ID"`
	SPUId            int64  `gorm:"not null;comment:SPU自增ID"`
	SKUId            int64  `gorm:"not null;index:idx_sku_id,comment:SKU自增ID"`
	SKUName          string `gorm:"type:varchar(255);not null;comment:SKU名称"`
	SKUDescription   string `gorm:"not null;comment:SKU描述"`
	SKUOriginalPrice int64  `gorm:"not null;comment:商品原始单价;单位为分, 999表示9.99元"`
	SKURealPrice     int64  `gorm:"not null;comment:商品实付单价;单位为分, 999表示9.99元"`
	Quantity         int64  `gorm:"not null;comment:购买数量"`
	Ctime            int64
	Utime            int64
}
