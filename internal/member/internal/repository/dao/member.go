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
	ErrDuplicatedMemberRecord = errors.New("会员记录重复")
	ErrCreateMemberConfict    = errors.New("创建会员主记录冲突")
	ErrUpdateMemberConflict   = errors.New("更新会员主记录冲突")
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
		// 1. first
		//    找到数据 快路径, 大部分情况,能找到数据, 即 update,
		//    未找到数据, create, 冲突, 重试,
		//
		for {
			err := g.upsert(tx, d, r)
			if errors.Is(err, ErrCreateMemberConfict) ||
				errors.Is(err, ErrUpdateMemberConflict) {
				continue
			}
			if err != nil {
				return err
			}
		}
	})
}

func (g *memberGROMDAO) upsert(tx *egorm.Component, d Member, r MemberRecord) error {

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, time.UTC)

	member := Member{
		EndAt:   g.endAt(today, r.Days),
		Version: 1,
		Ctime:   now.UnixMilli(),
		Utime:   now.UnixMilli(),
	}
	res := tx.Where(Member{Uid: d.Uid}).Attrs(member).FirstOrCreate(&member)
	if res.Error != nil {
		// case1: 同一个用户A, 两个不同的消息(Key不同) —— 注册福利会员和购买一月会员的请求到达
		// 两个协程并发消费,
		// g1, 执行FirstOrCreate,没找到,create,
		// g2, 执行FirstOrCreate,没找到,create的瞬间发现g1已创建,应该返回错误
		//     上层重试,使g2处理的请求走下方更新会员主记录(续约流程) 因为从结果上看应该会员截止日期是叠加的
		// g1和g2都会创建会员流水记录
		// case2: 同一个用户A, 两个相同的消息(Key相同) —— 注册福利会员和注册福利会员
		// 两个协程并发消费,
		// g1, 执行FirstOrCreate,没找到,create,
		// g2, 执行FirstOrCreate,没找到,create的瞬间发现g1已创建,应该返回错误
		//     上层重试,使g2处理的请求走下方更新(续约流程),此时走续约是错误,幸好在创建会员流水记录的时候会失败
		//     因为Key相同导致唯一索引冲突,导致整合g2重试的事务失败
		if g.isMySQLUniqueIndexError(res.Error) {
			return fmt.Errorf("%w", ErrCreateMemberConfict)
		}
		return res.Error
	}
	if res.RowsAffected == 0 {
		// 更新主记录
		if member.EndAt < now.UnixMilli() {
			// 重新激活
			member.EndAt = g.endAt(today, r.Days)
		} else {
			// 续约
			member.EndAt = g.endAt(time.UnixMilli(member.EndAt), r.Days)
		}
		member.Version += 1
		member.Utime = now.UnixMilli()
		res = tx.Model(&Member{}).
			Where("uid = ? AND version = ?", member.Uid, member.Version-1).Updates(&member)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// case: version被其他并发事务更新 通知上层重试
			return fmt.Errorf("%w", ErrUpdateMemberConflict)
		}
	}
	// 创建会员记录
	r.Uid = d.Uid
	r.Ctime, r.Utime = now.UnixMilli(), now.UnixMilli()
	if err := tx.Create(&r).Error; err != nil {
		if g.isMySQLUniqueIndexError(err) {
			return fmt.Errorf("%w", ErrDuplicatedMemberRecord)
		}
		return err
	}
	return nil
}

func (g *memberGROMDAO) endAt(startAt time.Time, days uint64) int64 {
	return startAt.Add(time.Hour * 24 * time.Duration(days)).UnixMilli()
}

func (g *memberGROMDAO) isMySQLUniqueIndexError(err error) bool {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		const uniqueIndexErrNo uint16 = 1062
		if me.Number == uniqueIndexErrNo {
			return true
		}
	}
	return false
}

// Member 会员表,每个用户只有一条记录,后续只需要修改开始、结束日期及状态即可
type Member struct {
	Id      int64 `gorm:"primaryKey;autoIncrement;comment:会员表自增ID"`
	Uid     int64 `gorm:"not null;uniqueIndex:unq_user_id;comment: 用户ID"`
	EndAt   int64 `gorm:"not null;comment: 会员结束日期,UTC Unix毫秒数"`
	Version int64 `gorm:"not null;default:1;comment: 版本号"`
	Ctime   int64
	Utime   int64
}

// MemberRecord 会员记录表 每次开通、激活、续约的流水记录
type MemberRecord struct {
	Id    int64  `gorm:"primaryKey;autoIncrement;comment:会员记录表自增ID"`
	Key   string `gorm:"type:varchar(256);not null;uniqueIndex:unq_key;comment:去重key"`
	Uid   int64  `gorm:"not null;index:idx_user_id;comment:用户ID"`
	Biz   string `gorm:"type:varchar(256);not null;comment:业务类型名,项目中模块目录名小写,user/member"`
	BizId int64  `gorm:"not null;index:idx_biz_id;comment:业务ID"`
	Desc  string `gorm:"type:varchar(256);not null;comment:会员流水描述"`
	Days  uint64 `gorm:"not null;comment:会员天数"`
	Ctime int64
	Utime int64
}
