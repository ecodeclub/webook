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
)

var (
	ErrUpdateMemberFailed     = errors.New("更新会员主记录失败")
	ErrDuplicatedMemberRecord = errors.New("会员记录重复")
)

type MemberDAO interface {
	FindMemberByUID(ctx context.Context, uid int64) (Member, error)
	FindMemberRecordsByUID(ctx context.Context, uid int64) ([]MemberRecord, error)
	Upsert(ctx context.Context, d Member, r MemberRecord) error
}

type memberGROMDAO struct {
	db *egorm.Component
}

func NewMemberGORMDAO(db *egorm.Component) MemberDAO {
	return &memberGROMDAO{db: db}
}

func (g *memberGROMDAO) FindMemberByUID(ctx context.Context, uid int64) (Member, error) {
	var m Member
	err := g.db.WithContext(ctx).First(&m, "uid", uid).Error
	return m, err
}

func (g *memberGROMDAO) FindMemberRecordsByUID(ctx context.Context, uid int64) ([]MemberRecord, error) {
	var r []MemberRecord
	err := g.db.WithContext(ctx).Order("ctime DESC").Find(&r, "uid", uid).Error
	return r, err
}

func (g *memberGROMDAO) Upsert(ctx context.Context, d Member, r MemberRecord) error {

	return g.db.WithContext(ctx).Transaction(func(tx *egorm.Component) error {

		// 新增
		now := time.Now().UTC()
		startAtDate := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)
		startAt := startAtDate.UnixMilli()
		endAt := startAtDate.Add(time.Hour * 24 * time.Duration(r.Days)).UnixMilli()

		member := Member{
			StartAt: startAt,
			EndAt:   endAt,
			Version: 1,
			Ctime:   now.UnixMilli(),
			Utime:   now.UnixMilli(),
		}
		res := tx.Where(Member{Uid: d.Uid}).Attrs(member).FirstOrCreate(&member)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// 更新主记录
			if member.EndAt < now.UnixMilli() {
				// 激活
				member.StartAt = startAt
				member.EndAt = endAt
			} else {
				// 续约
				endAt = time.UnixMilli(member.EndAt).Add(time.Hour * 24 * time.Duration(r.Days)).UnixMilli()
				member.EndAt = endAt
			}
			member.Version += 1
			member.Utime = now.UnixMilli()
			res = tx.Model(&Member{}).
				Where("uid = ? AND version = ?", member.Uid, member.Version-1).Updates(&member)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				// version被其他并发事务更新
				return fmt.Errorf("更新会员主记录失败: %w", ErrUpdateMemberFailed)
			}
		}
		// 创建会员记录
		r.Uid = d.Uid
		r.Ctime, r.Utime = now.UnixMilli(), now.UnixMilli()
		if err := tx.Create(&r).Error; err != nil {
			var me *mysql.MySQLError
			if errors.As(err, &me) {
				const uniqueIndexErrNo uint16 = 1062
				if me.Number == uniqueIndexErrNo {
					return ErrDuplicatedMemberRecord
				}
			}
			return err
		}
		return nil

	})
}

// Member 会员表,每个用户只有一条记录,后续只需要修改开始、结束日期及状态即可
// todo: StartAt可以去掉
type Member struct {
	Id      int64 `gorm:"primaryKey;autoIncrement;comment:会员表自增ID"`
	Uid     int64 `gorm:"not null;uniqueIndex:unq_user_id;comment: 用户ID"`
	StartAt int64 `gorm:"not null;comment: 会员开始日期,UTC Unix毫秒数"`
	EndAt   int64 `gorm:"not null;comment: 会员结束日期,UTC Unix毫秒数"`
	Version int64 `gorm:"not null;default:1;comment: 版本号"`
	Ctime   int64
	Utime   int64
}

// MemberRecord
// todo: Uid 和 Member.Id效果一样, 都是唯一的, 用Uid更方便查询
// todo: 用固定的order_id, type还是更抽象的 Biz(也是int64 or string), BizId, Desc 比如: 兑换优惠券、他人赠送、自己购买
// todo: Days 始终就是 正整数, 表示增加的天数, Desc字段有问题描述: 月会员 * 1, Days描述增加的天数
// todo: Ctime表示开会员的时间
type MemberRecord struct {
	Id  int64  `gorm:"primaryKey;autoIncrement;comment:会员流水表自增ID"`
	Key string `gorm:"type:varchar(256);not null;uniqueIndex:unq_key;comment:去重key"`
	Uid int64  `gorm:"not null;index:idx_user_id;comment:用户ID"`
	// `order_id` CHAR(16) NOT NULL UNIQUE COMMENT '订单ID, orders.id',
	// `type` TINYINT UNSIGNED NOT NULL COMMENT '记录类型, 赠送, 自己购买',
	Biz   int64  `gorm:"type:tinyint unsigned;not null;default:1;comment:业务类型 1=注册 2=购买"`
	BizId int64  `gorm:"not null;index:idx_biz_id;comment:业务ID"`
	Desc  string `gorm:"type:varchar(256);not null;comment:会员流水描述"`
	Days  uint64 `gorm:"not null;comment:会员天数"`
	Ctime int64
	Utime int64
}
