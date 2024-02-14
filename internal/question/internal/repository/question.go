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

package repository

import (
	"context"
	"time"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/repository/dao"
)

type Repository interface {
	Save(ctx context.Context, que *domain.Question) (int64, error)
}

type CachedRepository struct {
	dao dao.QuestionDAO
}

func NewCacheRepository(d dao.QuestionDAO) Repository {
	return &CachedRepository{
		dao: d,
	}
}

func (c *CachedRepository) Save(ctx context.Context, que *domain.Question) (int64, error) {
	q, eles := c.toEntity(que)
	return c.dao.Save(ctx, q, eles)
}

func (c *CachedRepository) toEntity(que *domain.Question) (dao.Question, []dao.AnswerElement) {
	now := time.Now().UnixMilli()
	q := dao.Question{
		Id:      que.Id,
		Uid:     que.Uid,
		Title:   que.Title,
		Content: que.Content,
		Ctime:   now,
		Utime:   now,
	}
	// 固定是 4 个部分
	eles := []dao.AnswerElement{
		c.ele2Entity(que.Id, now, dao.AnswerElementTypeAnalysis, que.Answer.Analysis),
		c.ele2Entity(que.Id, now, dao.AnswerElementTypeBasic, que.Answer.Basic),
		c.ele2Entity(que.Id, now, dao.AnswerElementTypeIntermedia, que.Answer.Intermediate),
		c.ele2Entity(que.Id, now, dao.AnswerElementTypeAdvanced, que.Answer.Advanced),
	}
	return q, eles
}

func (c *CachedRepository) ele2Entity(qid int64,
	now int64,
	typ uint8,
	ele domain.AnswerElement) dao.AnswerElement {
	return dao.AnswerElement{
		Qid:       qid,
		Type:      typ,
		Content:   ele.Content,
		Highlight: ele.Highlight,
		Keywords:  ele.Keywords,
		Shorthand: ele.Shorthand,
		Guidance:  ele.Guidance,
		Ctime:     now,
		Utime:     now,
	}
}
