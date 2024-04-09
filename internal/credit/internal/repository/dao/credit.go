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

	"github.com/ecodeclub/webook/internal/credit/internal/domain"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

var (
	ErrRecordChangedCuncurrently = errors.New("记录已被并发修改")
)

type CreditDAO interface {
	Upsert(ctx context.Context, uid int64, amount uint64, l CreditLog) (int64, error)
	FindCreditByUID(ctx context.Context, uid int64) (Credit, error)
	FindCreditLogsByUID(ctx context.Context, uid int64) ([]CreditLog, error)
	CreateCreditLockLog(ctx context.Context, uid int64, amount uint64, l CreditLog) (int64, error)
	ConfirmCreditLockLog(ctx context.Context, uid, tid int64) error
	CancelCreditLockLog(ctx context.Context, uid, tid int64) error
}

type creditDAO struct {
	db *egorm.Component
}

func NewCreditGORMDAO(db *egorm.Component) CreditDAO {
	return &creditDAO{db: db}
}

func (g *creditDAO) Upsert(ctx context.Context, uid int64, amount uint64, l CreditLog) (int64, error) {
	var cid int64
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		c := Credit{TotalCredits: amount, Version: 1, Ctime: now, Utime: now}
		res := tx.Where(Credit{Uid: uid}).Attrs(c).FirstOrCreate(&c)
		if res.RowsAffected == 0 {
			// 找到积分主记录, 更新可用积分
			version := c.Version
			c.TotalCredits += amount
			c.Version += 1
			c.Utime = now
			if err := tx.Model(&Credit{}).
				Where("uid = ? AND Version = ?", uid, version).
				Updates(map[string]any{
					"TotalCredits": c.TotalCredits, // 更新后可能为0
					"Utime":        c.Utime,
					"Version":      c.Version,
				}).Error; err != nil {
				return fmt.Errorf("更新积分主记录失败: %w", err)
			}
		}
		// 添加积分流水记录
		cid = c.Id
		l.Uid = uid
		l.CreditChange = int64(amount)
		l.CreditBalance = c.TotalCredits
		l.Ctime = now
		l.Utime = now
		if err := tx.Create(&l).Error; err != nil {
			return err
		}
		return nil
	})
	return cid, err
}

func (g *creditDAO) FindCreditByUID(ctx context.Context, uid int64) (Credit, error) {
	var res Credit
	err := g.db.WithContext(ctx).First(&res, "uid = ?", uid).Error
	return res, err
}

func (g *creditDAO) FindCreditLogsByUID(ctx context.Context, uid int64) ([]CreditLog, error) {
	var res []CreditLog
	err := g.db.WithContext(ctx).
		Where("uid = ? AND status = ?", uid, domain.CreditLogStatusActive).
		Order("ctime DESC").
		Find(&res).Error
	return res, err
}

// CreateCreditLockLog 创建积分预扣记录
func (g *creditDAO) CreateCreditLockLog(ctx context.Context, uid int64, amount uint64, l CreditLog) (int64, error) {
	var lid int64
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		now := time.Now().UnixMilli()

		var c Credit
		if err := tx.First(&c, "uid = ?", uid).Error; err != nil {
			return fmt.Errorf("积分主记录不存在: %w", err)
		}

		// 找到积分主记录, 更新可用积分
		version := c.Version
		c.TotalCredits -= amount
		c.LockedTotalCredits += amount
		c.Version += 1
		c.Utime = now
		if err := tx.Model(&Credit{}).
			Where("uid = ? AND Version = ?", uid, version).
			Updates(map[string]any{
				"TotalCredits":       c.TotalCredits,       // 更新后可能为0
				"LockedTotalCredits": c.LockedTotalCredits, // 更新后可能为0
				"Utime":              c.Utime,
				"Version":            c.Version,
			}).Error; err != nil {
			return fmt.Errorf("更新积分主记录失败: %w", err)
		}

		// 添加积分流水记录
		l.Uid = c.Uid
		l.CreditChange = 0 - int64(amount)
		l.CreditBalance = c.TotalCredits
		l.Status = domain.CreditLogStatusLocked
		l.Ctime = now
		l.Utime = now
		if err := tx.Create(&l).Error; err != nil {
			return err
		}
		lid = l.Id
		return nil
	})
	return lid, err
}

func (g *creditDAO) ConfirmCreditLockLog(ctx context.Context, uid, tid int64) error {
	res := g.db.WithContext(ctx).
		Model(&CreditLog{}).
		Where("uid = ? AND id = ? AND status = ?", uid, tid, domain.CreditLogStatusLocked).
		Updates(map[string]any{
			"Status": domain.CreditLogStatusActive,
			"Utime":  time.Now().UnixMilli(),
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("事务ID非法")
	}
	return nil
}

func (g *creditDAO) CancelCreditLockLog(ctx context.Context, uid, tid int64) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新
		now := time.Now().UnixMilli()

		var c Credit
		if err := tx.WithContext(ctx).First(&c, "uid = ?", uid).Error; err != nil {
			return fmt.Errorf("用户ID非法: %w", err)
		}

		var cl CreditLog
		if err := tx.WithContext(ctx).
			Where("uid = ? AND id = ? AND status = ?", uid, tid, domain.CreditLogStatusLocked).
			First(&cl).Error; err != nil {
			return fmt.Errorf("事务ID非法: %w", err)
		}

		cl.Status = domain.CreditLogStatusInactive
		cl.Utime = now
		if err := tx.WithContext(ctx).Model(&CreditLog{}).
			Where("uid = ? AND id = ? AND status = ?", uid, tid, domain.CreditLogStatusLocked).
			Updates(cl).Error; err != nil {
			return fmt.Errorf("更新积分流水记录失败: %w", err)
		}

		changeMount := uint64(0 - cl.CreditChange)
		version := c.Version
		c.TotalCredits += changeMount
		c.LockedTotalCredits -= changeMount
		c.Version += 1
		c.Utime = now
		if err := tx.Model(&Credit{}).
			Where("uid = ? AND Version = ?", uid, version).
			Updates(map[string]any{
				"TotalCredits":       c.TotalCredits, // 更新后可能为0
				"LockedTotalCredits": c.LockedTotalCredits,
				"Utime":              c.Utime,
				"Version":            c.Version,
			}).Error; err != nil {
			return fmt.Errorf("更新积分主记录失败: %w", err)
		}

		return nil
	})
}

type Credit struct {
	Id                 int64  `gorm:"primaryKey;autoIncrement;comment:积分主表自增ID"`
	Uid                int64  `gorm:"not null;uniqueIndex:unq_user_id;comment:用户ID"`
	TotalCredits       uint64 `gorm:"not null;default 0;comment:可用的积分总数"`
	LockedTotalCredits uint64 `gorm:"not null;default 0;comment:锁定的积分总数"`
	Version            int64  `gorm:"not null;default 1;comment:版本号"`
	Ctime              int64
	Utime              int64
}

type CreditLog struct {
	Id            int64  `gorm:"primaryKey;autoIncrement;comment:积分流水表自增ID"`
	Key           string `gorm:"type:varchar(256);not null;uniqueIndex:unq_key;comment:去重key"`
	Uid           int64  `gorm:"not null;index:idx_user_id;comment:用户ID"`
	Biz           int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:业务类型 1=注册 2=购买"`
	BizId         int64  `gorm:"not null;index:idx_biz_id;comment:业务ID"`
	Desc          string `gorm:"type:varchar(256);not null;comment:积分流水描述"`
	CreditChange  int64  `gorm:"not null;comment:积分变动数量,正数为增加,负数为减少"`
	CreditBalance uint64 `gorm:"not null;comment:变动后可用的积分总数"`
	Status        int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:流水状态 1=已生效, 2=预扣中, 3=已失效"`
	Ctime         int64
	Utime         int64
}
