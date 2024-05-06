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

	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ego-component/egorm"
)

type ProjectDAO interface {
	List(ctx context.Context, offset int, limit int) ([]PubProject, error)
	GetById(ctx context.Context, id int64) (PubProject, error)
	Resumes(ctx context.Context, pid int64) ([]PubProjectResume, error)
	Difficulties(ctx context.Context, pid int64) ([]PubProjectDifficulty, error)
	Questions(ctx context.Context, pid int64) ([]PubProjectQuestion, error)
	Introductions(ctx context.Context, pid int64) ([]PubProjectIntroduction, error)
}

var _ ProjectDAO = &GORMProjectDAO{}

type GORMProjectDAO struct {
	db *egorm.Component
}

func (dao *GORMProjectDAO) Introductions(ctx context.Context, pid int64) ([]PubProjectIntroduction, error) {
	var res []PubProjectIntroduction
	err := dao.db.WithContext(ctx).Where("pid = ?", pid).Find(&res).Error
	return res, err
}

func (dao *GORMProjectDAO) GetById(ctx context.Context, id int64) (PubProject, error) {
	var res PubProject
	err := dao.db.WithContext(ctx).Where("id = ? AND status = ?",
		id, domain.ProjectStatusPublished.ToUint8()).First(&res).Error
	return res, err
}

func (dao *GORMProjectDAO) Resumes(ctx context.Context, pid int64) ([]PubProjectResume, error) {
	var res []PubProjectResume
	err := dao.db.WithContext(ctx).
		Where("pid = ? AND status = ?",
			pid, domain.ResumeStatusPublished.ToUint8()).Find(&res).Error
	return res, err
}

func (dao *GORMProjectDAO) Difficulties(ctx context.Context, pid int64) ([]PubProjectDifficulty, error) {
	var res []PubProjectDifficulty
	err := dao.db.WithContext(ctx).
		Where("pid = ? AND status = ?",
			pid, domain.DifficultyStatusPublished.ToUint8()).Find(&res).Error
	return res, err
}

func (dao *GORMProjectDAO) Questions(ctx context.Context, pid int64) ([]PubProjectQuestion, error) {
	var res []PubProjectQuestion
	err := dao.db.WithContext(ctx).
		Where("pid = ? AND status = ?",
			pid, domain.QuestionStatusPublished.ToUint8()).Find(&res).Error
	return res, err
}

func (dao *GORMProjectDAO) List(ctx context.Context, offset int, limit int) ([]PubProject, error) {
	var res []PubProject
	err := dao.db.WithContext(ctx).Where("status = ?",
		domain.ProjectStatusPublished.ToUint8()).
		Order("utime DESC").
		Limit(limit).Offset(offset).Find(&res).Error
	return res, err
}

func NewGORMProjectDAO(db *egorm.Component) ProjectDAO {
	return &GORMProjectDAO{db: db}
}
