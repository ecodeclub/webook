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

	"github.com/ecodeclub/ekit/slice"

	"github.com/ego-component/egorm"
	"gorm.io/gorm"
)

type QuestionSetDAO interface {
	Create(ctx context.Context, qs QuestionSet) (int64, error)
	GetByID(ctx context.Context, id int64) (QuestionSet, error)

	GetQuestionsByID(ctx context.Context, id int64) ([]Question, error)
	GetPubQuestionsByID(ctx context.Context, id int64) ([]PublishQuestion, error)

	UpdateQuestionsByID(ctx context.Context, id int64, qids []int64) error
	CountByBiz(ctx context.Context, biz string) (int64, error)
	Count(ctx context.Context) (int64, error)
	List(ctx context.Context, offset, limit int) ([]QuestionSet, error)
	UpdateNonZero(ctx context.Context, set QuestionSet) error
	GetByIDs(ctx context.Context, ids []int64) ([]QuestionSet, error)
	ListByBiz(ctx context.Context, offset int, limit int, biz string) ([]QuestionSet, error)
	GetByBiz(ctx context.Context, biz string, bizId int64) (QuestionSet, error)
	// GetByIDsWithQuestions 返回题集和题集对应的题目id
	GetByIDsWithQuestions(ctx context.Context, ids []int64) ([]QuestionSet, map[int64][]Question, error)
}

type GORMQuestionSetDAO struct {
	db *egorm.Component
}

func (g *GORMQuestionSetDAO) CountByBiz(ctx context.Context, biz string) (int64, error) {
	var res int64
	db := g.db.WithContext(ctx).Model(&QuestionSet{})
	err := db.Select("COUNT(id)").
		Where("biz = ?", biz).
		Count(&res).Error
	return res, err
}

func (g *GORMQuestionSetDAO) GetByIDsWithQuestions(ctx context.Context, ids []int64) ([]QuestionSet, map[int64][]Question, error) {
	var sets []QuestionSet
	db := g.db.WithContext(ctx)
	err := db.Where("id IN ?", ids).Find(&sets).Error
	if err != nil {
		return nil, nil, err
	}
	var questions []QuestionSetQuestion
	err = db.Model(&QuestionSetQuestion{}).Where("qs_id IN ?", ids).Find(&questions).Error
	if err != nil {
		return nil, nil, err
	}
	qids := slice.Map(questions, func(idx int, src QuestionSetQuestion) int64 {
		return src.QID
	})
	var qs []Question
	err = db.Model(&Question{}).Where("id IN ?", qids).Find(&qs).Error
	if err != nil {
		return nil, nil, err
	}
	qmap := slice.ToMap(qs, func(element Question) int64 {
		return element.Id
	})
	questionMap := make(map[int64][]Question, len(ids))
	for _, q := range questions {
		questionList, ok := questionMap[q.QSID]
		question := qmap[q.QID]
		if ok {
			questionList = append(questionList, question)
			questionMap[q.QSID] = questionList
		} else {
			questionMap[q.QSID] = []Question{
				question,
			}
		}
	}
	return sets, questionMap, nil
}

func (g *GORMQuestionSetDAO) GetByBiz(ctx context.Context, biz string, bizId int64) (QuestionSet, error) {
	var res QuestionSet
	db := g.db.WithContext(ctx)
	err := db.Where("biz = ? AND biz_id = ?", biz, bizId).
		Order("utime DESC").
		First(&res).Error
	return res, err
}

func (g *GORMQuestionSetDAO) ListByBiz(ctx context.Context, offset int, limit int, biz string) ([]QuestionSet, error) {
	var res []QuestionSet
	db := g.db.WithContext(ctx)
	err := db.Where("biz = ?", biz).
		Offset(offset).Limit(limit).Order("id DESC").Find(&res).Error
	return res, err
}

func (g *GORMQuestionSetDAO) GetByIDs(ctx context.Context, ids []int64) ([]QuestionSet, error) {
	var res []QuestionSet
	err := g.db.WithContext(ctx).Where("id IN ?", ids).Find(&res).Error
	return res, err
}

func (g *GORMQuestionSetDAO) UpdateNonZero(ctx context.Context, set QuestionSet) error {
	set.Utime = time.Now().UnixMilli()
	return g.db.WithContext(ctx).Where("id = ?", set.Id).Updates(set).Error
}

func (g *GORMQuestionSetDAO) Create(ctx context.Context, qs QuestionSet) (int64, error) {
	qs.Ctime = qs.Utime
	err := g.db.WithContext(ctx).Create(&qs).Error
	if err != nil {
		return 0, err
	}
	return qs.Id, err
}

func (g *GORMQuestionSetDAO) GetByID(ctx context.Context, id int64) (QuestionSet, error) {
	var qs QuestionSet
	if err := g.db.WithContext(ctx).First(&qs, "id = ?", id).Error; err != nil {
		return QuestionSet{}, err
	}
	return qs, nil
}

func (g *GORMQuestionSetDAO) GetQuestionsByID(ctx context.Context, id int64) ([]Question, error) {
	var qsq []QuestionSetQuestion
	tx := g.db.WithContext(ctx)
	if err := tx.Find(&qsq, "qs_id = ?", id).Error; err != nil {
		return nil, err
	}
	questionIDs := make([]int64, len(qsq))
	for i := range qsq {
		questionIDs[i] = qsq[i].QID
	}
	var q []Question
	err := tx.WithContext(ctx).Where("id IN ?", questionIDs).Order("id ASC").Find(&q).Error
	return q, err
}

func (g *GORMQuestionSetDAO) GetPubQuestionsByID(ctx context.Context, id int64) ([]PublishQuestion, error) {
	var qsq []QuestionSetQuestion
	tx := g.db.WithContext(ctx)
	if err := tx.Find(&qsq, "qs_id = ?", id).Error; err != nil {
		return nil, err
	}
	questionIDs := make([]int64, len(qsq))
	for i := range qsq {
		questionIDs[i] = qsq[i].QID
	}
	var q []PublishQuestion
	err := tx.WithContext(ctx).Where("id IN ?", questionIDs).Order("id ASC").Find(&q).Error
	return q, err
}

func (g *GORMQuestionSetDAO) UpdateQuestionsByID(ctx context.Context, id int64, qids []int64) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var qs QuestionSet
		if err := tx.WithContext(ctx).First(&qs, "id = ? ", id).Error; err != nil {
			return err
		}
		// 全部删除
		if err := tx.WithContext(ctx).Where("qs_id = ?", id).Delete(&QuestionSetQuestion{}).Error; err != nil {
			return err
		}

		if len(qids) == 0 {
			return nil
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
		return tx.WithContext(ctx).Create(&newQuestions).Error
	})
}

func (g *GORMQuestionSetDAO) Count(ctx context.Context) (int64, error) {
	var res int64
	db := g.db.WithContext(ctx).Model(&QuestionSet{})
	err := db.Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (g *GORMQuestionSetDAO) List(ctx context.Context, offset, limit int) ([]QuestionSet, error) {
	var res []QuestionSet
	db := g.db.WithContext(ctx)
	err := db.Offset(offset).Limit(limit).Order("id DESC").Find(&res).Error
	return res, err
}

func NewGORMQuestionSetDAO(db *egorm.Component) QuestionSetDAO {
	return &GORMQuestionSetDAO{db: db}
}
