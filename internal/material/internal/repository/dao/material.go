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

// Status 定义了素材的状态枚举
type Status string

const (
	StatusInit     Status = "INIT"
	StatusAccepted Status = "ACCEPTED"
)

type Material struct {
	ID        int64  `gorm:"primaryKey,autoIncrement"`
	Uid       int64  `gorm:"NOT NULL;index;comment:'上传用户的ID'"`
	AudioURL  string `gorm:"type:VARCHAR(512);NOT NULL;comment:'面试录音的URL'"`
	ResumeURL string `gorm:"type:VARCHAR(512);NOT NULL;comment:'面试简历的URL'"`
	Remark    string `gorm:"type:TEXT;comment:'备注'"`
	Status    string `gorm:"type:ENUM('INIT','ACCEPTED');NOT NULL;default:'INIT';index;comment:'素材状态'"`
	Ctime     int64
	Utime     int64
}

func (Material) TableName() string {
	return "materials"
}

// MaterialDAO 定义了素材模块的数据访问操作
type MaterialDAO interface {
	Create(ctx context.Context, m Material) (int64, error)
	FindByID(ctx context.Context, id int64) (Material, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	FindAll(ctx context.Context, offset, limit int) ([]Material, error)
	CountAll(ctx context.Context) (int64, error)
}

// GORMMaterialDAO 是 MaterialDAO 的 GORM 实现
type GORMMaterialDAO struct {
	db *egorm.Component
}

func NewGORMMaterialDAO(db *egorm.Component) MaterialDAO {
	return &GORMMaterialDAO{db: db}
}

func (g *GORMMaterialDAO) Create(ctx context.Context, m Material) (int64, error) {
	now := time.Now().UnixMilli()
	m.Ctime = now
	m.Utime = now
	err := g.db.WithContext(ctx).Create(&m).Error
	return m.ID, err
}

func (g *GORMMaterialDAO) FindByID(ctx context.Context, id int64) (Material, error) {
	var material Material
	err := g.db.WithContext(ctx).Where("id = ?", id).First(&material).Error
	return material, err
}

func (g *GORMMaterialDAO) UpdateStatus(ctx context.Context, id int64, status string) error {
	return g.db.WithContext(ctx).Model(&Material{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status": status,
			"utime":  time.Now().UnixMilli(),
		}).Error
}

func (g *GORMMaterialDAO) FindAll(ctx context.Context, offset, limit int) ([]Material, error) {
	var res []Material
	err := g.db.WithContext(ctx).Model(&Material{}).
		Order("id DESC").
		Offset(offset).
		Limit(limit).
		Find(&res).Error
	return res, err
}

func (g *GORMMaterialDAO) CountAll(ctx context.Context) (int64, error) {
	var total int64
	err := g.db.WithContext(ctx).Model(&Material{}).Count(&total).Error
	return total, err
}
