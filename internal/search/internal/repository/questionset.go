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

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
)

type questionSetRepo struct {
	qsDao dao.QuestionSetDAO
}

func NewQuestionSetRepo(questionSetDao dao.QuestionSetDAO) QuestionSetRepo {
	return &questionSetRepo{
		qsDao: questionSetDao,
	}
}
func (q *questionSetRepo) SearchQuestionSet(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]domain.QuestionSet, error) {
	sets, err := q.qsDao.SearchQuestionSet(ctx, offset, limit, queryMetas)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.QuestionSet, 0, len(sets))
	for _, set := range sets {
		ans = append(ans, q.toDomain(set))
	}

	return ans, nil
}

func (*questionSetRepo) toDomain(qs *dao.QuestionSet) domain.QuestionSet {
	return domain.QuestionSet{
		Id:    qs.Id,
		Uid:   qs.Uid,
		Title: qs.Title,
		Biz:   qs.Biz,
		BizID: qs.BizID,
		Description: domain.EsVal{
			Val:           qs.Description,
			HighLightVals: qs.EsHighLights["description"],
		},
		Questions: qs.Questions,
		Utime:     time.UnixMilli(qs.Utime),
	}
}
