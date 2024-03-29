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
	"fmt"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ego-component/egorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type QuestionDAO interface {
	Update(ctx context.Context, q Question, eles []AnswerElement) error
	Create(ctx context.Context, q Question, eles []AnswerElement) (int64, error)
	GetByID(ctx context.Context, id int64) (Question, []AnswerElement, error)
	List(ctx context.Context, offset int, limit int, uid int64) ([]Question, error)
	Count(ctx context.Context, uid int64) (int64, error)

	Sync(ctx context.Context, que Question, eles []AnswerElement) (int64, error)

	// 线上库 API
	PubList(ctx context.Context, offset int, limit int) ([]PublishQuestion, error)
	PubCount(ctx context.Context) (int64, error)
	GetPubByID(ctx context.Context, qid int64) (PublishQuestion, []PublishAnswerElement, error)
}

type GORMQuestionDAO struct {
	db *egorm.Component
}

func (g *GORMQuestionDAO) GetPubByID(ctx context.Context, qid int64) (PublishQuestion, []PublishAnswerElement, error) {
	var q PublishQuestion
	db := g.db.WithContext(ctx)
	err := db.Where("id = ?", qid).First(&q).Error
	if err != nil {
		return PublishQuestion{}, nil, err
	}
	var eles []PublishAnswerElement
	// 总体上，只有四条数据，所以排序不会有什么性能问题
	err = db.Where("qid = ?", qid).Order("type ASC").Find(&eles).Error
	return q, eles, err
}

func (g *GORMQuestionDAO) Update(ctx context.Context, q Question, eles []AnswerElement) error {
	now := time.Now().UnixMilli()
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Question{}).WithContext(ctx).Where("id = ? AND uid = ?", q.Id, q.Uid).Updates(map[string]any{
			"title":   q.Title,
			"content": q.Content,
			"utime":   now,
		})
		if res.Error != nil {
			return res.Error
		}
		// 没有更新到数据，说明非法访问
		if res.RowsAffected < 1 {
			return fmt.Errorf("非法访问资源 uid %d, id %d", q.Uid, q.Id)
		}
		return g.saveEles(tx, eles)
	})
}

func (g *GORMQuestionDAO) update(tx *gorm.DB, q Question, eles []AnswerElement) error {
	now := time.Now().UnixMilli()
	res := tx.Model(&q).Where("id = ? AND uid = ?", q.Id, q.Uid).Updates(map[string]any{
		"title":   q.Title,
		"content": q.Content,
		"utime":   now,
	})
	if res.Error != nil {
		return res.Error
	}
	// 没有更新到数据，说明非法访问
	if res.RowsAffected < 1 {
		return fmt.Errorf("非法访问资源 uid %d, id %d", q.Uid, q.Id)
	}
	return g.saveEles(tx, eles)
}

func (g *GORMQuestionDAO) Create(ctx context.Context, q Question, eles []AnswerElement) (int64, error) {
	var qid int64
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		qid, err = g.create(tx, q, eles)
		return err
	})
	return qid, err
}

func (g *GORMQuestionDAO) create(tx *gorm.DB, q Question, eles []AnswerElement) (int64, error) {
	err := tx.Create(&q).Error
	if err != nil {
		return 0, err
	}
	qid := q.Id
	for i := range eles {
		eles[i].Qid = qid
	}
	return qid, g.saveEles(tx, eles)
}

func (g *GORMQuestionDAO) List(ctx context.Context, offset int, limit int, uid int64) ([]Question, error) {
	var res []Question
	err := g.db.WithContext(ctx).Where("uid = ?", uid).
		Offset(offset).Limit(limit).
		Order("id DESC").
		Find(&res).Error
	return res, err
}

func (g *GORMQuestionDAO) Count(ctx context.Context, uid int64) (int64, error) {
	var res int64
	err := g.db.WithContext(ctx).Model(&Question{}).Where("uid = ?", uid).Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (g *GORMQuestionDAO) PubList(ctx context.Context, offset int, limit int) ([]PublishQuestion, error) {
	var res []PublishQuestion
	err := g.db.WithContext(ctx).Offset(offset).
		Limit(limit).Order("id DESC").
		Find(&res).Error
	return res, err
}

func (g *GORMQuestionDAO) PubCount(ctx context.Context) (int64, error) {
	var res int64
	err := g.db.WithContext(ctx).Model(&PublishQuestion{}).Select("COUNT(id)").Count(&res).Error
	return res, err
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

func (g *GORMQuestionDAO) saveEles(tx *gorm.DB, eles []AnswerElement) error {
	return tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{
			"content", "keywords", "utime",
			"shorthand", "highlight", "guidance"}),
	}).Create(&eles).Error
}

func (g *GORMQuestionDAO) saveLive(tx *gorm.DB, que PublishQuestion, eles []PublishAnswerElement) error {
	res := tx.Save(&que)
	if res.Error != nil {
		return res.Error
	}
	err := tx.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{
			"content", "keywords", "utime",
			"shorthand", "highlight", "guidance"}),
	}).Create(&eles).Error
	return err
}

func (g *GORMQuestionDAO) Sync(ctx context.Context, que Question, eles []AnswerElement) (int64, error) {
	qid := que.Id
	err := g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if que.Id > 0 {
			err = g.update(tx, que, eles)
		} else {
			qid, err = g.create(tx, que, eles)
		}
		if err != nil {
			return err
		}
		pubEles := slice.Map(eles, func(idx int, src AnswerElement) PublishAnswerElement {
			return PublishAnswerElement(src)
		})
		return g.saveLive(tx, PublishQuestion(que), pubEles)
	})
	return qid, err
}

func NewGORMQuestionDAO(db *egorm.Component) QuestionDAO {
	return &GORMQuestionDAO{db: db}
}
