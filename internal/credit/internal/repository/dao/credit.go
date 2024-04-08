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
	"errors"
	"fmt"
	"time"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

var (
	ErrRecordChangedCuncurrently = errors.New("记录已被并发修改")
)

type CreditDAO interface {
	FindCreditByUID(ctx context.Context, uid int64) (Credit, error)
	Create(ctx context.Context, c Credit, l CreditLog) (int64, error)
	Update(ctx context.Context, c Credit, l CreditLog) error
}

type creditDAO struct {
	db *egorm.Component
}

func NewCreditGORMDAO(db *egorm.Component) CreditDAO {
	return &creditDAO{db: db}
}

func (g *creditDAO) FindCreditByUID(ctx context.Context, uid int64) (Credit, error) {
	var res Credit
	err := g.db.WithContext(ctx).First(&res, "uid = ?", uid).Error
	return res, err
}

func (g *creditDAO) Create(ctx context.Context, c Credit, l CreditLog) (int64, error) {
	var id int64
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		c.Utime, c.Ctime, l.Utime, l.Ctime = now, now, now, now
		if err := tx.Create(&c).Error; err != nil {
			return err
		}
		id = c.Id
		l.Cid = id
		if err := tx.Create(&l).Error; err != nil {
			return err
		}
		return nil
	})
	return id, err
}

func (g *creditDAO) Upsert(ctx context.Context, uid, amount int64, l CreditLog) (int64, error) {
	var id int64
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		var c Credit
		res := tx.Where(Credit{Uid: uid}).Attrs(Credit{TotalCredits: amount, Ctime: now, Utime: now}).FirstOrCreate(&c)
		if res.RowsAffected == 0 {
			// 找到积分主记录, 更新可用积分
			version := c.Version
			c.TotalCredits += amount
			c.Version += 1
			c.Utime = now
			if err := tx.Model(&Credit{}).
				Where("uid = ? AND Version = ?", uid, version).
				// Select("TotalCredits", "Utime", "Version").
				Updates(map[string]any{
					"TotalCredits": c.TotalCredits, // 更新后可能为0
					"Utime":        now,
					"Version":      c.Version,
				}).Error; err != nil {
				return fmt.Errorf("更新积分失败: %w", err)
			}
		}
		// 添加积分流水记录
		id = c.Id
		l.Cid = id
		l.CreditChange = amount
		l.CreditBalance = c.TotalCredits
		l.Ctime = now
		l.Utime = now
		if err := tx.Create(&l).Error; err != nil {
			return err
		}
		return nil
	})
	return id, err
}

func (g *creditDAO) Update(ctx context.Context, c Credit, l CreditLog) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		currentVersion := c.Version
		c.Version++

		now := time.Now().UnixMilli()
		c.Utime = now

		if err := tx.Model(&Credit{}).
			Select("TotalCredits", "LockedTotalCredits", "Utime", "Version").
			Where("uid = ? AND Version = ?", c.Uid, currentVersion).Updates(&c).Error; err != nil {
			return fmt.Errorf("更新积分失败: %w", err)
		}

		l.Utime = now
		if err := tx.Create(&l).Error; err != nil {
			return fmt.Errorf("创建积分流水记录失败: %w", err)
		}
		return nil
	})
}

/*
// CreateCreditLock 创建积分预扣记录
func (g *creditDAO) CreateCreditLock(ctx context.Context, c domain.CreditLog) error {

	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 查询用户积分记录
		var credit Credit
		if err := tx.Where("user_id = ?", c.UserId).First(&credit).Error; err != nil {
			return err
		}

		// 检查积分是否足够
		balance := credit.TotalCredits - credit.LockedTotalCredits - c.Amount
		if balance < 0 {
			return fmt.Errorf("积分不足")
		}

		// 增加预扣总积分
		// version := credit.Version
		// credit.Version++
		credit.LockedTotalCredits += c.Amount

		// todo: 创建一个未生效流水记录
		// 将计算部分移动到service层
		creditLog := CreditLog{
			CreditChange:  c.Amount,
			CreditBalance: balance,
			Desc:          "购买商品",
			Type:          CreditLogTypeBuyProduct,
			Status:        2, // 支付未生效
		}
		if err := g.Update(ctx, credit, creditLog); err != nil {
			return fmt.Errorf("更新锁定积分并创建未生效积分流水失败: %w", err)
		}

		// 创建预扣记录
		now := time.Now().UnixMilli()
		c.Ctime, c.Utime = now, now
		if err := tx.Create(&c).Error; err != nil {
			return fmt.Errorf("创建预扣记录失败: %w", err)
		}

		return nil
	})
}

// DeductCredits 根据积分预扣记录ID真实扣减积分
func (g *creditDAO) DeductCredits(ctx context.Context, lockID int64, c Credit, l CreditLog) error {

	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 更新积分表并添加积分记录
		// todo: 1) 积分表,totalCredits和lockedTotalCredits都减去amount, 再将creditLog的记录的status=1改为生效
		// todo: 思考是否将creditLog与creditLock合并??
		if err := g.Update(ctx, c, l); err != nil {
			return fmt.Errorf("更新积分表并添加流水记录失败: %w", err)
		}

		// 更新预扣记录状态为已扣减
		updateData := map[string]any{
			// "Status": CreditLockStatusDeducted,
			"Utime": time.Now().UnixMilli(),
		}
		if err := tx.Where("id = ?", lockID).Updates(updateData).Error; err != nil {
			return err
		}

		return nil
	})
}
*/

const (
	CreditLogStatusActive   = 1
	CreditLogStatusLocked   = 2
	CreditLogStatusInactive = 3
)

type Credit struct {
	Id                 int64 `gorm:"primaryKey;autoIncrement;comment:积分主表自增ID"`
	Uid                int64 `gorm:"not null;uniqueIndex:unq_user_id,comment:用户ID"`
	TotalCredits       int64 `gorm:"not null;default 0;comment:可用的积分总数"`
	LockedTotalCredits int64 `gorm:"not null;default 0;comment:锁定的积分总数"`
	Version            int64 `gorm:"not null;default 1;comment:版本号"`
	Ctime              int64
	Utime              int64
}

type CreditLog struct {
	Id            int64  `gorm:"primaryKey;autoIncrement;comment:积分流水表自增ID"`
	Cid           int64  `gorm:"not null;index:idx_credit_id,comment:积分主记录ID"`
	BizId         int64  `gorm:"not null;index:idx_biz_id,comment:业务ID"`
	BizType       int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:业务类型 1=注册 2=购买"`
	Desc          string `gorm:"type:varchar(255);not null;comment:积分流水描述"`
	CreditChange  int64  `gorm:"not null;comment:积分变动数量,正数为增加,负数为减少"`
	CreditBalance int64  `gorm:"not null;comment:变动后可用的积分总数"`
	Status        int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:流水状态 1=已生效, 2=预扣中, 3=已失效"`
	Ctime         int64
	Utime         int64
}
