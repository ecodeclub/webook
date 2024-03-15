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
)

type LabelDAO interface {
	UidLabels(ctx context.Context, uid int64) ([]Label, error)
}

type LabelGORMDAO struct {
	db *egorm.Component
}

func NewLabelGORMDAO(db *egorm.Component) LabelDAO {
	return &LabelGORMDAO{db: db}
}

func (dao *LabelGORMDAO) UidLabels(ctx context.Context, uid int64) ([]Label, error) {
	var res []Label
	err := dao.db.WithContext(ctx).
		Where("uid = ?", uid).Find(&res).Error
	return res, err
}

type Label struct {
	Id    int64 `gorm:"primaryKey,autoIncrement"`
	Name  string
	Uid   int64
	Ctime int64
	Utime int64
}
