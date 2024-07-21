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
	"gorm.io/gorm/clause"
)

type AdminDAO interface {
	Save(ctx context.Context, r Roadmap) (int64, error)
	GetById(ctx context.Context, id int64) (Roadmap, error)
	List(ctx context.Context, offset int, limit int) ([]Roadmap, error)
	GetEdgesByRid(ctx context.Context, rid int64) ([]Edge, error)
	AddEdge(ctx context.Context, edge Edge) error
	DeleteEdge(ctx context.Context, id int64) error
}

var _ AdminDAO = &GORMAdminDAO{}

type GORMAdminDAO struct {
	db *egorm.Component
}

func (dao *GORMAdminDAO) DeleteEdge(ctx context.Context, id int64) error {
	return dao.db.WithContext(ctx).Where("id = ?", id).Delete(&Edge{}).Error
}

func (dao *GORMAdminDAO) AddEdge(ctx context.Context, edge Edge) error {
	now := time.Now().UnixMilli()
	edge.Utime = now
	edge.Ctime = now
	return dao.db.WithContext(ctx).Create(&edge).Error
}

func (dao *GORMAdminDAO) GetEdgesByRid(ctx context.Context, rid int64) ([]Edge, error) {
	var res []Edge
	// 按照更新时间倒序排序
	err := dao.db.WithContext(ctx).Where("rid = ?", rid).Order("utime DESC").Find(&res).Error
	return res, err
}

func (dao *GORMAdminDAO) List(ctx context.Context, offset int, limit int) ([]Roadmap, error) {
	var res []Roadmap
	err := dao.db.WithContext(ctx).Order("id DESC").Offset(offset).Limit(limit).Find(&res).Error
	return res, err
}

func (dao *GORMAdminDAO) GetById(ctx context.Context, id int64) (Roadmap, error) {
	var r Roadmap
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&r).Error
	return r, err
}

func (dao *GORMAdminDAO) Save(ctx context.Context, r Roadmap) (int64, error) {
	now := time.Now().UnixMilli()
	r.Ctime = now
	r.Utime = now
	err := dao.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{"title", "biz", "biz_id", "utime"}),
		}).Create(&r).Error
	return r.Id, err
}

func NewGORMAdminDAO(db *egorm.Component) AdminDAO {
	return &GORMAdminDAO{
		db: db,
	}
}
