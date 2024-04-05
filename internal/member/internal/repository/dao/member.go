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
	"time"

	"github.com/ego-component/egorm"
)

var ErrUserDuplicate = errors.New("用户已有记录")

type MemberDAO interface {
	FindByUID(ctx context.Context, uid int64) (Member, error)
	Create(ctx context.Context, member Member) (int64, error)
}

type memberGROMDAO struct {
	db *egorm.Component
}

func NewMemberGORMDAO(db *egorm.Component) MemberDAO {
	return &memberGROMDAO{db: db}
}

func (g *memberGROMDAO) FindByUID(ctx context.Context, uid int64) (Member, error) {
	var m Member
	err := g.db.WithContext(ctx).First(&m, "uid", uid).Error
	return m, err
}

func (g *memberGROMDAO) Create(ctx context.Context, member Member) (int64, error) {
	now := time.Now().UnixMilli()
	member.Ctime, member.Utime = now, now
	if err := g.db.WithContext(ctx).Create(&member).Error; err != nil {
		return 0, err
	}
	return member.Id, nil
}

// Member 会员表,每个用户只有一条记录,后续只需要修改开始、结束日期及状态即可
type Member struct {
	Id      int64 `gorm:"primaryKey;autoIncrement;comment:会员表自增ID"`
	Uid     int64 `gorm:"not null;uniqueIndex:unq_user_id;comment: 用户ID"`
	StartAt int64 `gorm:"not null;comment: 会员开始日期,2024-04-01 12:51:28"`
	EndAt   int64 `gorm:"not null;comment: 会员结束日期,2024-06-30 23:59:59"`
	Status  int64 `gorm:"type:tinyint unsigned;not null;default:1;comment:会员状态 1=有效, 2=失效"`
	Ctime   int64
	Utime   int64
}
