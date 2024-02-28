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
	"gorm.io/gorm"
)

var (
	ErrInvalidQuestionSetID = errors.New("题集ID非法")
	ErrInvalidQuestionID    = errors.New("问题ID非法")
)

type QuestionSetDAO interface {
	Create(ctx context.Context, qs QuestionSet) (int64, error)
	GetByIDAndUID(ctx context.Context, id, uid int64) (QuestionSet, error)

	GetQuestionsByID(ctx context.Context, id int64) ([]Question, error)
	UpdateQuestionsByIDAndUID(ctx context.Context, id, uid int64, qids []int64) error

	Count(ctx context.Context, uid int64) (int64, error)
	List(ctx context.Context, offset, limit int, uid int64) ([]QuestionSet, error)
}

type GORMQuestionSetDAO struct {
	db *egorm.Component
}

func NewGORMQuestionSetDAO(db *egorm.Component) QuestionSetDAO {
	return &GORMQuestionSetDAO{db: db}
}

func (g *GORMQuestionSetDAO) Create(ctx context.Context, qs QuestionSet) (int64, error) {
	qs.Ctime = qs.Utime
	err := g.db.WithContext(ctx).Create(&qs).Error
	if err != nil {
		return 0, err
	}
	return qs.Id, err
}

func (g *GORMQuestionSetDAO) GetByIDAndUID(ctx context.Context, id, uid int64) (QuestionSet, error) {
	var qs QuestionSet
	if err := g.db.WithContext(ctx).Find(&qs, "id = ? AND uid = ?", id, uid).Error; err != nil {
		return QuestionSet{}, err
	}
	if qs.Id == 0 {
		return QuestionSet{}, fmt.Errorf("%w", ErrInvalidQuestionSetID)
	}
	return qs, nil
}

func (g *GORMQuestionSetDAO) GetQuestionsByID(ctx context.Context, id int64) ([]Question, error) {
	var q []Question
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var qsq []QuestionSetQuestion
		if err := tx.WithContext(ctx).Find(&qsq, "qs_id = ?", id).Error; err != nil {
			return err
		}
		questionIDs := make([]int64, len(qsq))
		for i := range qsq {
			questionIDs[i] = qsq[i].QID
		}
		return tx.WithContext(ctx).Where("id IN ?", questionIDs).Order("id ASC").Find(&q).Error
	})
	return q, err
}

func (g *GORMQuestionSetDAO) UpdateQuestionsByIDAndUID(ctx context.Context, id, uid int64, qids []int64) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var qs QuestionSet
		if err := tx.WithContext(ctx).Find(&qs, "id = ? AND uid = ?", id, uid).Error; err != nil {
			return err
		}
		if qs.Id == 0 {
			return fmt.Errorf("%w", ErrInvalidQuestionSetID)
		}

		// 全部删除
		if err := tx.WithContext(ctx).Where("qs_id = ?", id).Delete(&QuestionSetQuestion{}).Error; err != nil {
			return err
		}

		if len(qids) == 0 {
			return nil
		}

		// 检查问题ID合法性
		var count int64
		if err := tx.WithContext(ctx).Model(&Question{}).Where("id IN ?", qids).Count(&count).Error; err != nil {
			return err
		}
		if int64(len(qids)) != count {
			return fmt.Errorf("%w", ErrInvalidQuestionID)
		}

		// 重新创建
		now := time.Now().UnixMilli()
		var newQuestions []QuestionSetQuestion
		for i := range qids {
			newQuestions = append(newQuestions, QuestionSetQuestion{
				QSID:  id,
				QID:   qids[i],
				Ctime: now,
				Utime: now,
			})
		}
		if err := tx.WithContext(ctx).Create(&newQuestions).Error; err != nil {
			return err
		}
		return nil
	})
}

func (g *GORMQuestionSetDAO) Count(ctx context.Context, uid int64) (int64, error) {
	var res int64
	db := g.db.WithContext(ctx).Model(&QuestionSet{})
	if uid != 0 {
		db = db.Where("uid = ?", uid)
	}
	err := db.Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (g *GORMQuestionSetDAO) List(ctx context.Context, offset, limit int, uid int64) ([]QuestionSet, error) {
	var res []QuestionSet
	db := g.db.WithContext(ctx)
	if uid != 0 {
		db = db.Where("uid = ?", uid)
	}
	err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&res).Error
	return res, err
}
