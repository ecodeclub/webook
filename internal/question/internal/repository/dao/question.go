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
	"gorm.io/gorm/clause"
)

type QuestionDAO interface {
	Save(ctx context.Context, que Question, eles []AnswerElement) (int64, error)
	GetByID(ctx context.Context, id int64) (Question, []AnswerElement, error)
}

type GORMQuestionDAO struct {
	db *egorm.Component
}

func (g *GORMQuestionDAO) GetByID(ctx context.Context, id int64) (Question, []AnswerElement, error) {
	var q Question
	db := g.db.WithContext(ctx)
	err := db.Where("id = ?", id).First(&q).Error
	if err != nil {
		return Question{}, nil, err
	}
	var eles []AnswerElement
	// 总体上，只有四条数据，所以排序不会有什么性能问题
	err = db.Where("qid = ?", id).Order("type ASC").Find(&eles).Error
	return q, eles, err
}

func (g *GORMQuestionDAO) Save(ctx context.Context, que Question, eles []AnswerElement) (int64, error) {
	qid := que.Id
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Save(&que)
		if res.Error != nil {
			return res.Error
		}
		// 也就是插入，而不是更新
		if qid == 0 {
			qid = que.Id
			for i := range eles {
				eles[i].Qid = que.Id
			}
		}
		for _, ele := range eles {
			err := g.db.Clauses(clause.OnConflict{
				DoUpdates: clause.AssignmentColumns([]string{
					"content", "keywords", "utime",
					"shorthand", "highlight", "guidance"}),
			}).Create(&ele).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	return qid, err
}

func NewGORMQuestionDAO(db *egorm.Component) QuestionDAO {
	return &GORMQuestionDAO{db: db}
}
