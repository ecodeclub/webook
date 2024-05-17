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

	"github.com/ego-component/egorm"
)

type PermissionDAO interface {
	CreatePersonalPermission(ctx context.Context, ps []PersonalPermission) error
	CountPersonalPermission(ctx context.Context, p PersonalPermission) (int64, error)
	FindPersonalPermissions(ctx context.Context, uid int64) ([]PersonalPermission, error)
}

type gormPermissionDAO struct {
	db *egorm.Component
}

func NewPermissionGORMDAO(db *egorm.Component) PermissionDAO {
	return &gormPermissionDAO{db: db}
}

func (g *gormPermissionDAO) CreatePersonalPermission(ctx context.Context, ps []PersonalPermission) error {
	now := time.Now().UnixMilli()
	return g.db.WithContext(ctx).Transaction(func(tx *egorm.Component) error {
		for _, p := range ps {
			if err := tx.Where(PersonalPermission{Uid: p.Uid, Biz: p.Biz, BizId: p.BizId}).
				Attrs(PersonalPermission{Desc: p.Desc, Ctime: now, Utime: now}).FirstOrCreate(&p).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (g *gormPermissionDAO) CountPersonalPermission(ctx context.Context, p PersonalPermission) (int64, error) {
	var count int64
	result := g.db.WithContext(ctx).Model(&PersonalPermission{}).
		Where("uid = ? AND biz = ? AND biz_id = ?", p.Uid, p.Biz, p.BizId).Count(&count)
	return count, result.Error
}

func (g *gormPermissionDAO) FindPersonalPermissions(ctx context.Context, uid int64) ([]PersonalPermission, error) {
	var res []PersonalPermission
	err := g.db.WithContext(ctx).Model(&PersonalPermission{}).Where("uid = ?", uid).
		Order("Ctime DESC").Find(&res).Error
	return res, err
}

type PersonalPermission struct {
	Id    int64  `gorm:"primaryKey;autoIncrement;comment:个人权限自增ID"`
	Uid   int64  `gorm:"not null;uniqueIndex:uniq_uid_biz_biz_id;comment:用户ID"`
	Biz   string `gorm:"type:varchar(255);not null;uniqueIndex:uniq_uid_biz_biz_id;comment:业务名称, project"`
	BizId int64  `gorm:"not null;uniqueIndex:uniq_uid_biz_biz_id;comment:业务实体ID"`
	Desc  string `gorm:"type:varchar(256);not null;comment:权限描述"`
	Ctime int64
	Utime int64
}
