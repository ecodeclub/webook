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

	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"gorm.io/gorm"
)

type PaymentDAO interface {
	Insert(ctx context.Context, pmt Payment) error
	UpdateTxnIDAndStatus(ctx context.Context, bizTradeNo string, txnID string, status int64) error
	FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]Payment, error)
	GetPayment(ctx context.Context, bizTradeNO string) (Payment, error)
}

type PaymentGORMDAO struct {
	db *gorm.DB
}

func NewPaymentGORMDAO(db *gorm.DB) PaymentDAO {
	return &PaymentGORMDAO{db: db}
}

func (p *PaymentGORMDAO) GetPayment(ctx context.Context, bizTradeNO string) (Payment, error) {
	var res Payment
	err := p.db.WithContext(ctx).Where("biz_trade_no = ?", bizTradeNO).First(&res).Error
	return res, err
}

func (p *PaymentGORMDAO) FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]Payment, error) {
	var res []Payment
	err := p.db.WithContext(ctx).Where("status = ? AND utime < ?", domain.PaymentStatusUnpaid, t.UnixMilli()).Offset(offset).Limit(limit).Find(&res).Error
	return res, err
}

func (p *PaymentGORMDAO) UpdateTxnIDAndStatus(ctx context.Context,
	bizTradeNo string,
	txnID string, status int64) error {
	return p.db.WithContext(ctx).Model(&Payment{}).
		Where("biz_trade_no = ?", bizTradeNo).
		Updates(map[string]any{
			"txn_id": txnID,
			"status": status,
			"utime":  time.Now().UnixMilli(),
		}).Error
}

func (p *PaymentGORMDAO) Insert(ctx context.Context, pmt Payment) error {
	now := time.Now().UnixMilli()
	pmt.Utime = now
	pmt.Ctime = now
	return p.db.WithContext(ctx).Create(&pmt).Error
}

type Payment struct {
	Id               int64  `gorm:"primaryKey;autoIncrement;comment:支付自增ID"`
	SN               string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_payment_sn;comment:支付序列号"`
	OrderId          int64  `gorm:"uniqueIndex:uniq_order_id,comment:订单自增ID,冗余允许为NULL"`
	OrderSn          string `gorm:"type:varchar(255);uniqueIndex:uniq_order_sn;comment:订单序列号,冗余允许为NULL"`
	OrderDescription string `gorm:"type:varchar(255);not null;comment:订单简要描述"`
	TotalAmount      int64  `gorm:"not null;comment:支付总金额, 多种支付方式支付金额的总和"`
	PayDDL           int64  `gorm:"column:pay_ddl;not null;comment:支付截止时间"`
	PaidAt           int64  `gorm:"comment:支付时间"`
	Status           int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:支付状态 1=未支付 2=已支付"`
	Ctime            int64
	Utime            int64
}

type PaymentRecord struct {
	Id           int64  `gorm:"primaryKey;autoIncrement;comment:支付记录自增ID"`
	PaymentId    int64  `gorm:"not null;index:idx_payment_id,comment:支付自增ID"`
	PaymentNO3rd string `gorm:"column:payment_no_3rd;type:varchar(255);not null;uniqueIndex:uniq_payment_no_3rd;comment:支付单号, 支付渠道的事务ID"`
	Channel      int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:支付渠道 1=积分, 2=微信"`
	Amount       int64  `gorm:"not null;comment:支付金额"`
	PaidAt       int64  `gorm:"comment:支付时间"`
	Status       int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:支付状态 1=未支付 2=已支付"`
	Ctime        int64
	Utime        int64
}
