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
	Create(ctx context.Context, c Credit, l CreditLog) error
	Update(ctx context.Context, c Credit, l CreditLog) error
	CreateCreditLock(ctx context.Context, c CreditLock) error
	FindByUID(ctx context.Context, uid int64) (Credit, error)
}

type creditDAO struct {
	db *egorm.Component
}

func NewGORMCreditDAO(db *egorm.Component) CreditDAO {
	return &creditDAO{db: db}
}

func (g *creditDAO) Create(ctx context.Context, c Credit, l CreditLog) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		c.Utime, c.Ctime, l.Utime, l.Ctime = now, now, now, now
		if err := tx.Create(&c).Error; err != nil {
			return err
		}
		if err := tx.Create(&l).Error; err != nil {
			return err
		}
		return nil
	})
}

func (g *creditDAO) Update(ctx context.Context, c Credit, l CreditLog) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		currentVersion := c.Version
		c.Version++

		now := time.Now().UnixMilli()
		c.Utime = now

		if err := tx.Model(&Credit{}).
			Select("TotalCredits", "LockedTotalCredits", "Utime", "Version").
			Where("user_id = ? AND Version = ?", c.UserId, currentVersion).Updates(&c).Error; err != nil {
			return fmt.Errorf("更新积分失败: %w", err)
		}

		l.Utime = now
		if err := tx.Create(&l).Error; err != nil {
			return fmt.Errorf("创建积分流水记录失败: %w", err)
		}
		return nil
	})
}

// CreateCreditLock 创建积分预扣记录
func (g *creditDAO) CreateCreditLock(ctx context.Context, c CreditLock) error {

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
			SN:            "SN--", // 返回给支付模块使用
			UserId:        c.UserId,
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
			"Status": CreditLockStatusDeducted,
			"Utime":  time.Now().UnixMilli(),
		}
		if err := tx.Model(&CreditLock{}).Where("id = ?", lockID).Updates(updateData).Error; err != nil {
			return err
		}

		return nil
	})
}

func (g *creditDAO) FindByUID(ctx context.Context, uid int64) (Credit, error) {
	var res Credit
	err := g.db.WithContext(ctx).First(&res, "user_id = ?", uid).Error
	return res, err
}

const (
	CreditLogTypeRegitser = iota + 1
	CreditLogTypeInviteOtherRegitser
	CreditLogTypeBuyProduct
)

const (
	CreditLockStatusLocked = iota + 1
	CreditLockStatusDeducted
	CreditLockStatusReleased
)

type Credit struct {
	Id                 int64 `gorm:"primaryKey;autoIncrement;comment:积分表自增ID"`
	UserId             int64 `gorm:"not null;uniqueIndex:unq_user_id,comment:用户ID"`
	TotalCredits       int64 `gorm:"not null;default 0;comment:积分总数"`
	LockedTotalCredits int64 `gorm:"not null;default 0;comment:锁定的积分总数"`
	Version            int64 `gorm:"not null;default 1;comment:版本号"`
	Ctime              int64
	Utime              int64
}

type CreditLog struct {
	Id            int64  `gorm:"primaryKey;autoIncrement;comment:积分流水表自增ID"`
	SN            string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_credit_log_sn;comment:积分流水序列号"`
	UserId        int64  `gorm:"not null;index:idx_user_id,comment:用户ID"`
	CreditChange  int64  `gorm:"not null;comment:积分变动数量,正数为获得,负数为消耗"`
	CreditBalance int64  `gorm:"not null;comment:变动后的积分余额"`
	Desc          string `gorm:"type:varchar(255);not null;comment:积分流水描述"`
	Type          int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:流水类型 1=首次注册 2=邀请他人注册 3=购买商品 4=邀请他人购买商品"`
	Status        int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:流水状态 1=未生效, 2=已生效"`
	Ctime         int64
	Utime         int64
}

type CreditLock struct {
	Id     int64 `gorm:"primaryKey;autoIncrement;comment:积分预扣记录表自增ID"`
	UserId int64 `gorm:"not null;index:idx_user_id,comment:用户ID"`
	Amount int64 `gorm:"not null;comment:预扣积分数"`
	Status int64 `gorm:"type:tinyint unsigned;not null;default:1;comment:预扣状态 1=预扣中 2=已扣减 3=已释放"`
	Ctime  int64
	Utime  int64
}
