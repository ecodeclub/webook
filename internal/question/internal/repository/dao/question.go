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
	"gorm.io/gorm/clause"
)

type QuestionDAO interface {
	Update(ctx context.Context, q Question, eles []AnswerElement) error
	Create(ctx context.Context, q Question, eles []AnswerElement) (int64, error)
	GetByID(ctx context.Context, id int64) (Question, []AnswerElement, error)
	List(ctx context.Context, offset int, limit int) ([]Question, error)
	Count(ctx context.Context) (int64, error)
	// 用于
	Ids(ctx context.Context) ([]int64, error)
	// Delete 会直接删除制作库和线上库的数据
	Delete(ctx context.Context, qid int64) error

	Sync(ctx context.Context, que Question, eles []AnswerElement) (int64, error)

	// 线上库 API
	PubList(ctx context.Context, offset int, limit int, biz string) ([]PublishQuestion, error)
	PubCount(ctx context.Context, biz string) (int64, error)
	GetPubByID(ctx context.Context, qid int64) (PublishQuestion, []PublishAnswerElement, error)
	GetPubByIDs(ctx context.Context, qids []int64) ([]PublishQuestion, error)
	NotInTotal(ctx context.Context, ids []int64) (int64, error)
	NotIn(ctx context.Context, ids []int64, offset int, limit int) ([]Question, error)
}

type GORMQuestionDAO struct {
	db *egorm.Component
}

func (g *GORMQuestionDAO) Ids(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := g.db.WithContext(ctx).
		Select("id").
		Model(&Question{}).
		Where("status = ? ", 2).
		Scan(&ids).Error
	return ids, err
}

func (g *GORMQuestionDAO) NotInTotal(ctx context.Context, ids []int64) (int64, error) {
	var res int64
	err := g.db.WithContext(ctx).
		Model(&Question{}).
		Where("id NOT IN ?", ids).Count(&res).Error
	return res, err
}

func (g *GORMQuestionDAO) NotIn(ctx context.Context, ids []int64, offset int, limit int) ([]Question, error) {
	var res []Question
	err := g.db.WithContext(ctx).
		Model(&Question{}).
		Where("id NOT IN ?", ids).Order("utime DESC").
		Offset(offset).Limit(limit).Find(&res).Error
	return res, err
}

func (g *GORMQuestionDAO) GetPubByIDs(ctx context.Context, qids []int64) ([]PublishQuestion, error) {
	var qs []PublishQuestion
	db := g.db.WithContext(ctx)
	err := db.Where("id IN ?", qids).Find(&qs).Error
	return qs, err
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

// Delete 会直接删除制作库和线上库的数据
func (g *GORMQuestionDAO) Delete(ctx context.Context, qid int64) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("id = ?", qid).Delete(&Question{}).Error
		if err != nil {
			return err
		}

		err = tx.Where("id =?", qid).Delete(&PublishQuestion{}).Error
		if err != nil {
			return err
		}

		err = tx.Where("qid = ?", qid).Delete(&AnswerElement{}).Error
		if err != nil {
			return err
		}

		err = tx.Where("qid = ?", qid).Delete(&PublishAnswerElement{}).Error
		if err != nil {
			return err
		}
		return tx.Where("qid = ?", qid).Delete(&QuestionSetQuestion{}).Error
	})
}

func (g *GORMQuestionDAO) Update(ctx context.Context, q Question, eles []AnswerElement) error {
	return g.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return g.update(tx, q, eles)
	})
}

func (g *GORMQuestionDAO) update(tx *gorm.DB, q Question, eles []AnswerElement) error {
	now := time.Now().UnixMilli()
	res := tx.Model(&q).Where("id = ?", q.Id).Updates(map[string]any{
		"title":   q.Title,
		"content": q.Content,
		"labels":  q.Labels,
		"status":  q.Status,
		"utime":   now,
		"biz":     q.Biz,
		"biz_id":  q.BizId,
	})
	if res.Error != nil {
		return res.Error
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

func (g *GORMQuestionDAO) List(ctx context.Context, offset int, limit int) ([]Question, error) {
	var res []Question
	err := g.db.WithContext(ctx).
		Offset(offset).Limit(limit).
		Order("id DESC").
		Find(&res).Error
	return res, err
}

func (g *GORMQuestionDAO) Count(ctx context.Context) (int64, error) {
	var res int64
	err := g.db.WithContext(ctx).Model(&Question{}).Select("COUNT(id)").Count(&res).Error
	return res, err
}

func (g *GORMQuestionDAO) PubList(ctx context.Context, offset int, limit int, biz string) ([]PublishQuestion, error) {
	var res []PublishQuestion
	err := g.db.WithContext(ctx).Offset(offset).
		Where("biz = ?", biz).
		Limit(limit).Order("id DESC").
		Find(&res).Error
	return res, err
}

func (g *GORMQuestionDAO) PubCount(ctx context.Context, biz string) (int64, error) {
	var res int64
	err := g.db.WithContext(ctx).
		Where("biz = ?", biz).
		Model(&PublishQuestion{}).Select("COUNT(id)").Count(&res).Error
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
			// 强制将 id 设置为 0。因为前面的 update 或者 upsert 触发了 update 的时候，
			// 即便是执行了更新，GIN 也会赋予一个 id，但是这个 id 是错误的 id。
			// 我们依赖于唯一索引来更新
			src.Id = 0
			src.Qid = qid
			return PublishAnswerElement(src)
		})
		return g.saveLive(tx, PublishQuestion(que), pubEles)
	})
	return qid, err
}

func NewGORMQuestionDAO(db *egorm.Component) QuestionDAO {
	return &GORMQuestionDAO{db: db}
}
