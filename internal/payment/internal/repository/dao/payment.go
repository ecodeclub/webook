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
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"gorm.io/gorm"
)

type PaymentDAO interface {
	FindOrCreate(ctx context.Context, pmt Payment, records []PaymentRecord) (Payment, []PaymentRecord, error)
	FindPaymentByID(ctx context.Context, pmtID int64) (Payment, []PaymentRecord, error)
	UpdateByOrderSN(ctx context.Context, pmt Payment, records []PaymentRecord) error
	FindPaymentByOrderSN(ctx context.Context, orderSN string) (Payment, []PaymentRecord, error)

	// 下方待重构

	FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]Payment, error)
}

type PaymentGORMDAO struct {
	db *gorm.DB
}

func NewPaymentGORMDAO(db *gorm.DB) PaymentDAO {
	return &PaymentGORMDAO{db: db}
}

func (g *PaymentGORMDAO) FindOrCreate(ctx context.Context, pmt Payment, records []PaymentRecord) (Payment, []PaymentRecord, error) {
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		pmt.Ctime, pmt.Utime = now, now
		if err := tx.FirstOrCreate(&pmt, "order_id = ? AND order_sn = ?", pmt.OrderId, pmt.OrderSn).Error; err != nil {
			return fmt.Errorf("创建支付主记录失败: %w", err)
		}
		for i := 0; i < len(records); i++ {
			records[i].PaymentId = pmt.Id
			records[i].Ctime, records[i].Utime = now, now
			if err := tx.FirstOrCreate(&records[i], "payment_id = ? AND channel = ?", records[i].PaymentId, records[i].Channel).Error; err != nil {
				return fmt.Errorf("创建支付渠道记录失败: %w", err)
			}
		}
		return nil
	})
	return pmt, records, err
}

func (g *PaymentGORMDAO) FindPaymentByID(ctx context.Context, pmtID int64) (Payment, []PaymentRecord, error) {
	var (
		eg      errgroup.Group
		pmt     Payment
		records []PaymentRecord
	)
	eg.Go(func() error {
		return g.db.WithContext(ctx).Where("id = ?", pmtID).First(&pmt).Error
	})
	eg.Go(func() error {
		return g.db.WithContext(ctx).Where("payment_id = ?", pmtID).Order("channel desc").Find(&records).Error
	})
	return pmt, records, eg.Wait()
}

func (g *PaymentGORMDAO) UpdateByOrderSN(ctx context.Context, pmt Payment, records []PaymentRecord) error {
	utime := time.Now().UnixMilli()
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		pmt.Utime = utime
		// 确保pmt中更新paidAt, status,
		if err := tx.Model(&Payment{}).Where("order_sn = ?", pmt.OrderSn).Updates(&pmt).Error; err != nil {
			return fmt.Errorf("更新支付主记录失败: %w", err)
		}

		if err := tx.First(&pmt, "order_sn = ?", pmt.OrderSn).Error; err != nil {
			return fmt.Errorf("查找支付主记录失败: %w", err)
		}

		for i := 0; i < len(records); i++ {
			records[i].Utime = utime
			// 	需要确保record中要更新, paidAt, status, paymentNo3rd
			if err := tx.Model(&PaymentRecord{}).Where("payment_id = ? AND Channel = ?", pmt.Id, records[i].Channel).Updates(&records[i]).Error; err != nil {
				return fmt.Errorf("更新支付记录表失败: %w", err)
			}
		}

		return nil
	})
}

func (g *PaymentGORMDAO) FindPaymentByOrderSN(ctx context.Context, orderSN string) (Payment, []PaymentRecord, error) {
	var pmt Payment
	var records []PaymentRecord
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&pmt, "order_sn = ?", orderSN).Error; err != nil {
			return fmt.Errorf("查找支付主记录失败: %w", err)
		}
		if err := tx.Find(&records, "payment_id = ?", pmt.Id).Error; err != nil {
			return fmt.Errorf("查找支付渠道记录失败: %w", err)
		}
		return nil
	})
	return pmt, records, err
}

func (g *PaymentGORMDAO) FindExpiredPayment(ctx context.Context, offset int, limit int, t time.Time) ([]Payment, error) {
	var res []Payment
	err := g.db.WithContext(ctx).Where("status = ? AND utime < ?", domain.PaymentStatusUnpaid.ToUnit8(), t.UnixMilli()).Offset(offset).Limit(limit).Find(&res).Error
	return res, err
}

type Payment struct {
	Id               int64          `gorm:"primaryKey;autoIncrement;comment:支付自增ID"`
	SN               string         `gorm:"type:varchar(255);not null;uniqueIndex:uniq_payment_sn;comment:支付序列号"`
	PayerId          int64          `gorm:"index:idx_payer_id,comment:支付者ID"`
	OrderId          int64          `gorm:"uniqueIndex:uniq_order_id,comment:订单自增ID,冗余允许为NULL"`
	OrderSn          sql.NullString `gorm:"type:varchar(255);uniqueIndex:uniq_order_sn;comment:订单序列号,冗余允许为NULL"`
	OrderDescription string         `gorm:"type:varchar(255);not null;comment:订单简要描述"`
	TotalAmount      int64          `gorm:"not null;comment:支付总金额, 多种支付方式支付金额的总和"`
	PaidAt           int64          `gorm:"comment:支付时间"`
	Status           uint8          `gorm:"type:tinyint unsigned;not null;default:1;comment:支付状态 1=未支付 2=已支付 3=已失败"`
	Ctime            int64
	Utime            int64
}

type PaymentRecord struct {
	Id           int64          `gorm:"primaryKey;autoIncrement;comment:支付记录自增ID"`
	PaymentId    int64          `gorm:"not null;uniqueIndex:unq_idx_payment_id_channel;comment:支付自增ID"`
	PaymentNO3rd sql.NullString `gorm:"column:payment_no_3rd;type:varchar(255);uniqueIndex:uniq_payment_no_3rd;comment:支付单号, 支付渠道的事务ID"`
	Description  string         `gorm:"type:varchar(255);not null;comment:本次支付的简要描述"`
	Channel      uint8          `gorm:"type:tinyint unsigned;not null;default:1;uniqueIndex:unq_idx_payment_id_channel;comment:支付渠道 1=积分, 2=微信"`
	Amount       int64          `gorm:"not null;comment:支付金额"`
	PaidAt       int64          `gorm:"comment:支付时间"`
	Status       uint8          `gorm:"type:tinyint unsigned;not null;default:1;comment:支付状态 1=未支付 2=处理中 3=支付成功 4=支付失败"`
	Ctime        int64
	Utime        int64
}
