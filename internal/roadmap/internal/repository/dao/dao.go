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

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

var ErrRecordNotFound = gorm.ErrRecordNotFound

type RoadmapDAO interface {
	GetEdgesByRid(ctx context.Context, rid int64) ([]Edge, error)
	GetByBiz(ctx context.Context, biz string, bizId int64) (Roadmap, error)
}

var _ RoadmapDAO = &GORMRoadmapDAO{}

type GORMRoadmapDAO struct {
	db *egorm.Component
}

func (dao *GORMRoadmapDAO) GetByBiz(ctx context.Context, biz string, bizId int64) (Roadmap, error) {
	var r Roadmap
	err := dao.db.WithContext(ctx).
		Where("biz = ? AND biz_id = ?", biz, bizId).
		First(&r).Error
	return r, err
}

func (dao *GORMRoadmapDAO) GetEdgesByRid(ctx context.Context, rid int64) ([]Edge, error) {
	var res []Edge
	err := dao.db.WithContext(ctx).Where("rid = ?", rid).Find(&res).Error
	return res, err
}

func NewGORMRoadmapDAO(db *egorm.Component) RoadmapDAO {
	return &GORMRoadmapDAO{db: db}
}
