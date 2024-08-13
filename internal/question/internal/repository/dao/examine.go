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

var ErrRecordNotFound = gorm.ErrRecordNotFound

type ExamineDAO interface {
	SaveResult(ctx context.Context, record ExamineRecord) error
	GetResultByUidAndQid(ctx context.Context, uid int64, qid int64) (QuestionResult, error)
	GetResultByUidAndQids(ctx context.Context, uid int64, ids []int64) ([]QuestionResult, error)
	UpdateQuestionResult(ctx context.Context, result QuestionResult) error
}

var _ ExamineDAO = &GORMExamineDAO{}

type GORMExamineDAO struct {
	db *egorm.Component
}

func (dao *GORMExamineDAO) GetResultByUidAndQids(ctx context.Context, uid int64, ids []int64) ([]QuestionResult, error) {
	var res []QuestionResult
	err := dao.db.WithContext(ctx).Where("uid = ? AND qid IN ?", uid, ids).Find(&res).Error
	return res, err
}

func (dao *GORMExamineDAO) GetResultByUidAndQid(ctx context.Context, uid, qid int64) (QuestionResult, error) {
	var res QuestionResult
	err := dao.db.WithContext(ctx).Where("uid = ? AND qid = ?", uid, qid).First(&res).Error
	return res, err
}

func (dao *GORMExamineDAO) SaveResult(ctx context.Context, record ExamineRecord) error {
	now := time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		record.Ctime = now
		record.Utime = now
		err := tx.Create(&record).Error
		if err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{
			// 如果有记录了，就更新结果和更新时间
			DoUpdates: clause.AssignmentColumns([]string{
				"result", "utime",
			}),
		}).Create(&QuestionResult{
			Uid:    record.Uid,
			Qid:    record.Qid,
			Result: record.Result,
			Ctime:  now,
			Utime:  now,
		}).Error
	})
}

func (dao *GORMExamineDAO) UpdateQuestionResult(ctx context.Context, result QuestionResult) error {
	now := time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&result).Where("uid = ? and qid = ?", result.Uid, result.Qid).Updates(map[string]any{
			"uid":    result.Uid,
			"qid":    result.Qid,
			"result": result.Result,
			"utime":  now,
		})
		return res.Error
	})
}

func NewGORMExamineDAO(db *egorm.Component) ExamineDAO {
	return &GORMExamineDAO{db: db}
}
