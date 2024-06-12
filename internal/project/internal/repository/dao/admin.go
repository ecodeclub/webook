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
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProjectAdminDAO interface {
	Save(ctx context.Context, prj Project) (int64, error)
	List(ctx context.Context, offset int, limit int) ([]Project, error)
	Count(ctx context.Context) (int64, error)
	GetById(ctx context.Context, id int64) (Project, error)
	// Sync 制作库同步到线上库
	Sync(ctx context.Context, entity Project) (int64, error)

	ResumeSave(ctx context.Context, resume ProjectResume) (int64, error)
	Resumes(ctx context.Context, pid int64) ([]ProjectResume, error)
	ResumeById(ctx context.Context, id int64) (ProjectResume, error)
	ResumeSync(ctx context.Context, rsm ProjectResume) (int64, error)

	DifficultySave(ctx context.Context, diff ProjectDifficulty) (int64, error)
	DifficultySync(ctx context.Context, diff ProjectDifficulty) (int64, error)
	Difficulties(ctx context.Context, pid int64) ([]ProjectDifficulty, error)
	DifficultyById(ctx context.Context, id int64) (ProjectDifficulty, error)

	QuestionSave(ctx context.Context, entity ProjectQuestion) (int64, error)
	QuestionById(ctx context.Context, id int64) (ProjectQuestion, error)
	Questions(ctx context.Context, pid int64) ([]ProjectQuestion, error)
	QuestionSync(ctx context.Context, que ProjectQuestion) (int64, error)

	IntroductionSave(ctx context.Context, intr ProjectIntroduction) (int64, error)
	IntroductionById(ctx context.Context, id int64) (ProjectIntroduction, error)
	IntroductionSync(ctx context.Context, intr ProjectIntroduction) (int64, error)
	Introductions(ctx context.Context, pid int64) ([]ProjectIntroduction, error)
	ComboSave(ctx context.Context, c ProjectCombo) (int64, error)
	ComboById(ctx context.Context, cid int64) (ProjectCombo, error)
	ComboSync(ctx context.Context, c ProjectCombo) (int64, error)
	Combos(ctx context.Context, pid int64) ([]ProjectCombo, error)
}

var _ ProjectAdminDAO = &GORMProjectAdminDAO{}

type GORMProjectAdminDAO struct {
	db               *egorm.Component
	prjUpdateColumns []string
}

func (dao *GORMProjectAdminDAO) Combos(ctx context.Context, pid int64) ([]ProjectCombo, error) {
	var res []ProjectCombo
	err := dao.db.WithContext(ctx).Where("pid = ?", pid).Find(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) ComboSync(ctx context.Context, c ProjectCombo) (int64, error) {
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := dao.comboSave(tx, &c)
		if err != nil {
			return err
		}
		pubCb := PubProjectCombo(c)
		return tx.Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{
				"content", "title", "status", "utime",
			}),
		}).Create(&pubCb).Error
	})
	return c.Id, err
}

func (dao *GORMProjectAdminDAO) ComboById(ctx context.Context, cid int64) (ProjectCombo, error) {
	var c ProjectCombo
	err := dao.db.WithContext(ctx).Where("id = ?", cid).First(&c).Error
	return c, err
}

func (dao *GORMProjectAdminDAO) ComboSave(ctx context.Context, c ProjectCombo) (int64, error) {
	return dao.comboSave(dao.db.WithContext(ctx), &c)
}

func (dao *GORMProjectAdminDAO) comboSave(tx *gorm.DB, c *ProjectCombo) (int64, error) {
	now := time.Now().UnixMilli()
	c.Utime = now
	c.Ctime = now
	err := tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{
			"content", "title", "status", "utime",
		}),
	}).Create(c).Error
	return c.Id, err
}

func (dao *GORMProjectAdminDAO) ResumeSync(ctx context.Context, rsm ProjectResume) (int64, error) {
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := dao.rsmSave(tx, &rsm)
		if err != nil {
			return err
		}
		pubRsm := PubProjectResume(rsm)
		return tx.Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{
				"role", "content", "analysis", "status", "utime",
			}),
		}).Create(&pubRsm).Error
	})
	return rsm.Id, err
}

func (dao *GORMProjectAdminDAO) QuestionSync(ctx context.Context, que ProjectQuestion) (int64, error) {
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := dao.queSave(tx, &que)
		if err != nil {
			return err
		}
		pubQue := PubProjectQuestion(que)
		return tx.Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{
				"title", "analysis", "answer", "status", "utime",
			}),
		}).Create(&pubQue).Error
	})
	return que.Id, err
}

func (dao *GORMProjectAdminDAO) DifficultySync(ctx context.Context, diff ProjectDifficulty) (int64, error) {
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		_, err := dao.diffSave(tx, &diff)
		if err != nil {
			return err
		}
		pubDiff := PubProjectDifficulty(diff)
		return tx.Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{
				"title", "content", "analysis", "status", "utime",
			}),
		}).Create(&pubDiff).Error
	})
	return diff.Id, err
}

func (dao *GORMProjectAdminDAO) Introductions(ctx context.Context, pid int64) ([]ProjectIntroduction, error) {
	var res []ProjectIntroduction
	err := dao.db.WithContext(ctx).Where("pid = ?", pid).Find(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) IntroductionSync(ctx context.Context, intr ProjectIntroduction) (int64, error) {
	id := intr.Id
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		id, err = dao.intrSave(tx, &intr)
		if err != nil {
			return err
		}
		pubIntr := PubProjectIntroduction(intr)
		return tx.Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns([]string{"role", "content",
				"analysis", "status", "utime"}),
		}).Create(&pubIntr).Error

	})
	return id, err
}

func (dao *GORMProjectAdminDAO) IntroductionById(ctx context.Context, id int64) (ProjectIntroduction, error) {
	var res ProjectIntroduction
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) IntroductionSave(ctx context.Context, intr ProjectIntroduction) (int64, error) {
	return dao.intrSave(dao.db.WithContext(ctx), &intr)
}

func (dao *GORMProjectAdminDAO) intrSave(tx *gorm.DB, intr *ProjectIntroduction) (int64, error) {
	now := time.Now().UnixMilli()
	intr.Utime = now
	intr.Ctime = now
	err := tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{"role", "content", "analysis", "status"}),
	}).Create(intr).Error
	return intr.Id, err
}

func (dao *GORMProjectAdminDAO) Sync(ctx context.Context, entity Project) (int64, error) {
	id := entity.Id
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		id, err = dao.save(tx, &entity)
		if err != nil {
			return err
		}
		pubEn := PubProject(entity)
		return tx.Clauses(clause.OnConflict{
			DoUpdates: clause.AssignmentColumns(dao.prjUpdateColumns),
		}).Create(&pubEn).Error
	})
	return id, err
}

func (dao *GORMProjectAdminDAO) Questions(ctx context.Context, pid int64) ([]ProjectQuestion, error) {
	var res []ProjectQuestion
	err := dao.db.WithContext(ctx).Where("pid = ?", pid).Find(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) QuestionById(ctx context.Context, id int64) (ProjectQuestion, error) {
	var res ProjectQuestion
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) QuestionSave(ctx context.Context, que ProjectQuestion) (int64, error) {
	return dao.queSave(dao.db.WithContext(ctx), &que)
}

func (dao *GORMProjectAdminDAO) queSave(tx *gorm.DB, que *ProjectQuestion) (int64, error) {
	now := time.Now().UnixMilli()
	que.Ctime = now
	que.Utime = now
	err := tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "analysis", "answer", "status", "utime",
		}),
	}).Create(&que).Error
	return que.Id, err
}

func (dao *GORMProjectAdminDAO) DifficultyById(ctx context.Context, id int64) (ProjectDifficulty, error) {
	var res ProjectDifficulty
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) Difficulties(ctx context.Context, pid int64) ([]ProjectDifficulty, error) {
	var res []ProjectDifficulty
	err := dao.db.WithContext(ctx).
		Where("pid = ?", pid).
		Order("utime DESC").
		Find(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) DifficultySave(ctx context.Context, diff ProjectDifficulty) (int64, error) {
	return dao.diffSave(dao.db.WithContext(ctx), &diff)
}

func (dao *GORMProjectAdminDAO) diffSave(tx *gorm.DB, diff *ProjectDifficulty) (int64, error) {
	now := time.Now().UnixMilli()
	diff.Utime = now
	diff.Ctime = now
	err := tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "content", "analysis", "status", "utime",
		}),
	}).Create(&diff).Error
	return diff.Id, err
}

func (dao *GORMProjectAdminDAO) ResumeById(ctx context.Context, id int64) (ProjectResume, error) {
	var res ProjectResume
	err := dao.db.WithContext(ctx).Where("id = ?", id).First(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) Resumes(ctx context.Context, pid int64) ([]ProjectResume, error) {
	var res []ProjectResume
	err := dao.db.WithContext(ctx).
		Where("pid = ?", pid).
		Order("utime DESC").
		Find(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) ResumeSave(ctx context.Context, resume ProjectResume) (int64, error) {
	return dao.rsmSave(dao.db.WithContext(ctx), &resume)
}

func (dao *GORMProjectAdminDAO) rsmSave(tx *gorm.DB, resume *ProjectResume) (int64, error) {
	now := time.Now().UnixMilli()
	resume.Utime = now
	resume.Ctime = now
	err := tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{
			"role", "content", "analysis", "status", "utime",
		}),
	}).Create(&resume).Error
	return resume.Id, err
}

func (dao *GORMProjectAdminDAO) GetById(ctx context.Context, id int64) (Project, error) {
	var res Project
	err := dao.db.WithContext(ctx).Where("id =?", id).First(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) Count(ctx context.Context) (int64, error) {
	var res int64
	err := dao.db.WithContext(ctx).Model(&Project{}).Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) List(ctx context.Context, offset int, limit int) ([]Project, error) {
	var res []Project
	err := dao.db.WithContext(ctx).
		Select("id", "sn", "title", "labels", "utime", "status", "desc").
		Offset(offset).Limit(limit).
		Order("utime DESC").
		Find(&res).Error
	return res, err
}

func (dao *GORMProjectAdminDAO) Save(ctx context.Context, prj Project) (int64, error) {
	return dao.save(dao.db.WithContext(ctx), &prj)
}

func (dao *GORMProjectAdminDAO) save(tx *gorm.DB, prj *Project) (int64, error) {
	now := time.Now().UnixMilli()
	prj.Ctime = now
	prj.Utime = now
	err := tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns(dao.prjUpdateColumns),
	}).Create(prj).Error
	return prj.Id, err
}

func NewGORMProjectAdminDAO(db *egorm.Component) *GORMProjectAdminDAO {
	return &GORMProjectAdminDAO{
		db: db,
		prjUpdateColumns: []string{
			"title", "status", "labels", "desc", "overview",
			"github_repo", "gitee_repo", "ref_question_set",
			"system_design", "utime"}}
}
