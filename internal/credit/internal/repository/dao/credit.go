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
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

var (
	ErrCreateCreditConflict         = errors.New("创建积分主记录冲突")
	ErrUpdateCreditConflict         = errors.New("更新积分主记录冲突")
	ErrDuplicatedCreditLog          = errors.New("积分流水记录重复")
	ErrCreditNotEnough              = errors.New("积分不足")
	ErrRecordNotFound               = egorm.ErrRecordNotFound
	ErrInvalidLockedCreditLogStatus = errors.New("锁定的积分流水初始状态非法")
)

type CreditDAO interface {
	Upsert(ctx context.Context, l CreditLog) error
	FindCreditByUID(ctx context.Context, uid int64) (Credit, error)
	FindCreditLogsByUID(ctx context.Context, uid int64) ([]CreditLog, error)
	CreateCreditLockLog(ctx context.Context, l CreditLog) (int64, error)
	ConfirmCreditLockLog(ctx context.Context, uid, tid int64) error
	CancelCreditLockLog(ctx context.Context, uid, tid int64) error
	FindExpiredLockedCreditLogs(ctx context.Context, offset int, limit int, ctime int64) ([]CreditLog, error)
	TotalExpiredLockedCreditLogs(ctx context.Context, ctime int64) (int64, error)
}

type creditDAO struct {
	db *egorm.Component
}

func NewCreditGORMDAO(db *egorm.Component) CreditDAO {
	return &creditDAO{db: db}
}

func (g *creditDAO) Upsert(ctx context.Context, l CreditLog) error {
	for {
		err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return g.upsert(tx, l)
		})
		if errors.Is(err, ErrCreateCreditConflict) || errors.Is(err, ErrUpdateCreditConflict) {
			continue
		}
		return err
	}
}

func (g *creditDAO) upsert(tx *gorm.DB, l CreditLog) error {
	now := time.Now().UnixMilli()
	uid := l.Uid
	amount := uint64(l.CreditChange)
	c := Credit{TotalCredits: amount, Version: 1, Ctime: now, Utime: now}
	res := tx.Where(Credit{Uid: uid}).Attrs(c).FirstOrCreate(&c)
	if res.Error != nil {
		if g.isMySQLUniqueIndexError(res.Error) {
			return fmt.Errorf("%w", ErrCreateCreditConflict)
		}
		return fmt.Errorf("创建/查找积分主记录失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		// 找到积分主记录, 更新可用积分
		version := c.Version
		c.TotalCredits += amount
		c.Version += 1
		c.Utime = now
		res = tx.Model(&Credit{}).
			Where("uid = ? AND Version = ?", uid, version).
			Updates(map[string]any{
				"TotalCredits": c.TotalCredits, // 更新后可能为0
				"Utime":        c.Utime,
				"Version":      c.Version,
			})
		if res.Error != nil {
			return fmt.Errorf("更新积分主记录失败: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			// case: version被其他并发事务更新 通知上层重试
			return fmt.Errorf("%w", ErrUpdateCreditConflict)
		}
	}
	// 添加积分流水记录
	l.CreditChange = int64(amount)
	l.CreditBalance = c.TotalCredits
	l.Ctime = now
	l.Utime = now
	if err := tx.Create(&l).Error; err != nil {
		if g.isMySQLUniqueIndexError(err) {
			return fmt.Errorf("%w", ErrDuplicatedCreditLog)
		}
		return err
	}
	return nil
}

func (g *creditDAO) isMySQLUniqueIndexError(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		const uniqueIndexErrNo uint16 = 1062
		if me.Number == uniqueIndexErrNo {
			return true
		}
	}
	return false
}

func (g *creditDAO) FindCreditByUID(ctx context.Context, uid int64) (Credit, error) {
	var res Credit
	err := g.db.WithContext(ctx).First(&res, "uid = ?", uid).Error
	return res, err
}

func (g *creditDAO) FindCreditLogsByUID(ctx context.Context, uid int64) ([]CreditLog, error) {
	var res []CreditLog
	err := g.db.WithContext(ctx).
		Where("uid = ? AND status != ?", uid, CreditLogStatusInactive).
		Order("ctime DESC").
		Find(&res).Error
	return res, err
}

// CreateCreditLockLog 创建积分预扣记录
func (g *creditDAO) CreateCreditLockLog(ctx context.Context, l CreditLog) (int64, error) {
	var lid int64
	for {
		err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			id, err := g.createCreditLockLog(tx, l)
			lid = id
			return err
		})
		if errors.Is(err, ErrUpdateCreditConflict) {
			continue
		}
		if errors.Is(err, ErrDuplicatedCreditLog) {
			return g.getCreditLogIDByKey(g.db, l.Key)
		}
		return lid, err
	}
}

func (g *creditDAO) createCreditLockLog(tx *gorm.DB, l CreditLog) (int64, error) {
	now := time.Now().UnixMilli()
	amount := uint64(l.CreditChange)

	var c Credit
	if err := tx.First(&c, "uid = ?", l.Uid).Error; err != nil {
		return 0, fmt.Errorf("积分主记录不存在: %w", err)
	}

	// 找到积分主记录, 更新可用积分
	version := c.Version
	if c.TotalCredits < amount {
		if id, err := g.getCreditLogIDByKey(tx, l.Key); err == nil {
			// 重复处理相同请求,返回第一次处理的结果
			return id, nil
		}
		return 0, fmt.Errorf("%w", ErrCreditNotEnough)
	}
	c.TotalCredits -= amount
	c.LockedTotalCredits += amount
	c.Version += 1
	c.Utime = now
	res := tx.Model(&Credit{}).
		Where("uid = ? AND Version = ?", l.Uid, version).
		Updates(map[string]any{
			"TotalCredits":       c.TotalCredits, // 更新后可能为0
			"LockedTotalCredits": c.LockedTotalCredits,
			"Utime":              c.Utime,
			"Version":            c.Version,
		})

	if err := res.Error; err != nil {
		return 0, fmt.Errorf("更新积分主记录失败: %w", err)
	}
	if res.RowsAffected == 0 {
		// case: version被其他并发事务更新 通知上层重试
		return 0, fmt.Errorf("%w", ErrUpdateCreditConflict)
	}

	// 添加积分流水记录
	l.CreditChange = 0 - int64(amount)
	l.CreditBalance = c.TotalCredits
	l.Status = CreditLogStatusLocked
	l.Ctime = now
	l.Utime = now
	if err := tx.Create(&l).Error; err != nil {
		if g.isMySQLUniqueIndexError(err) {
			return 0, fmt.Errorf("%w", ErrDuplicatedCreditLog)
		}
		return 0, err
	}
	return l.Id, nil
}

func (g *creditDAO) getCreditLogIDByKey(tx *gorm.DB, key string) (int64, error) {
	var cl CreditLog
	if err := tx.First(&cl, "`key` = ?", key).Error; err != nil {
		return 0, err
	}
	// 重复处理相同请求,返回第一次处理的结果
	return cl.Id, nil
}

// ConfirmCreditLockLog 确认预扣积分
func (g *creditDAO) ConfirmCreditLockLog(ctx context.Context, uid, tid int64) error {
	for {
		err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			totalCreditsIncreaseAmountFunc := func(cl CreditLog) uint64 { return uint64(0) }
			return g.updateCreditLockLog(tx, uid, tid, CreditLogStatusLocked, CreditLogStatusActive, totalCreditsIncreaseAmountFunc)
		})
		if errors.Is(err, ErrUpdateCreditConflict) {
			continue
		}
		return err
	}
}

// CancelCreditLockLog 取消积分预扣
func (g *creditDAO) CancelCreditLockLog(ctx context.Context, uid, tid int64) error {
	for {
		err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			totalCreditsIncreaseAmountFunc := func(cl CreditLog) uint64 { return uint64(0 - cl.CreditChange) }
			return g.updateCreditLockLog(tx, uid, tid, CreditLogStatusLocked, CreditLogStatusInactive, totalCreditsIncreaseAmountFunc)
		})
		if errors.Is(err, ErrUpdateCreditConflict) {
			continue
		}
		if errors.Is(err, ErrRecordNotFound) {
			return nil
		}
		return err
	}
}

func (g *creditDAO) updateCreditLockLog(tx *gorm.DB, uid, tid int64, srcStatus, dstStatus uint8,
	totalCreditsIncreaseAmountFunc func(cl CreditLog) uint64) error {
	// 更新
	now := time.Now().UnixMilli()

	var c Credit
	if err := tx.First(&c, "uid = ?", uid).Error; err != nil {
		return err
	}

	var cl CreditLog
	if err := tx.Where("uid = ? AND id = ?", uid, tid).First(&cl).Error; err != nil {
		return err
	}

	if cl.Status == dstStatus {
		// 已处理过并且达到目标状态 重复处理相同请求,返回第一次处理的结果
		return nil
	}

	if cl.Status != srcStatus {
		// 未达到预期初始状态
		return fmt.Errorf("%w: 已被修改为%d", ErrInvalidLockedCreditLogStatus, cl.Status)
	}

	cl.Status = dstStatus
	cl.Utime = now
	res := tx.Model(&CreditLog{}).
		Where("uid = ? AND id = ? AND status = ?", uid, tid, srcStatus).
		Updates(cl)
	if err := res.Error; err != nil {
		return fmt.Errorf("更新积分流水记录失败: %w", err)
	}
	if res.RowsAffected == 0 {
		// 并发修改
		return fmt.Errorf("%w", ErrUpdateCreditConflict)
	}

	version := c.Version
	c.TotalCredits += totalCreditsIncreaseAmountFunc(cl)
	c.LockedTotalCredits -= uint64(0 - cl.CreditChange)
	c.Version += 1
	c.Utime = now

	res = tx.Model(&Credit{}).
		Where("uid = ? AND Version = ?", uid, version).
		Updates(map[string]any{
			"TotalCredits":       c.TotalCredits,
			"LockedTotalCredits": c.LockedTotalCredits, // 更新后可能为0
			"Utime":              c.Utime,
			"Version":            c.Version,
		})
	if err := res.Error; err != nil {
		return fmt.Errorf("更新积分主记录失败: %w", err)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("%w", ErrUpdateCreditConflict)
	}
	return nil
}

func (g *creditDAO) FindExpiredLockedCreditLogs(ctx context.Context, offset int, limit int, ctime int64) ([]CreditLog, error) {
	var cs []CreditLog
	err := g.db.WithContext(ctx).
		Where("status = ? AND ctime <= ?", CreditLogStatusLocked, ctime).
		Offset(offset).Limit(limit).Order("ctime desc").Find(&cs).Error
	return cs, err
}

func (g *creditDAO) TotalExpiredLockedCreditLogs(ctx context.Context, ctime int64) (int64, error) {
	var res int64
	err := g.db.WithContext(ctx).Model(&CreditLog{}).
		Where("status = ? AND ctime <= ?", CreditLogStatusLocked, ctime).
		Select("COUNT(id)").Count(&res).Error
	return res, err
}

const (
	CreditLogStatusActive   uint8 = 1
	CreditLogStatusLocked   uint8 = 2
	CreditLogStatusInactive uint8 = 3
)

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
	Biz           string `gorm:"type:varchar(256);not null;comment:业务类型名,项目中模块目录名小写,user/member"`
	BizId         int64  `gorm:"not null;index:idx_biz_id;comment:业务ID"`
	Desc          string `gorm:"type:varchar(256);not null;comment:积分流水描述"`
	CreditChange  int64  `gorm:"not null;comment:积分变动数量,正数为增加,负数为减少"`
	CreditBalance uint64 `gorm:"not null;comment:变动后可用的积分总数"`
	Status        uint8  `gorm:"type:tinyint unsigned;not null;default:1;comment:流水状态 1=已生效, 2=已锁定, 3=已失效"`
	Ctime         int64
	Utime         int64
}
