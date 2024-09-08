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

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
)

type questionRepository struct {
	questionDao dao.QuestionDAO
}

func NewQuestionRepo(questionDao dao.QuestionDAO) QuestionRepo {
	return &questionRepository{
		questionDao: questionDao,
	}
}

func (q *questionRepository) SearchQuestion(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]domain.Question, error) {
	ques, err := q.questionDao.SearchQuestion(ctx, offset, limit, queryMetas)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.Question, 0, len(ques))
	for _, qu := range ques {
		ans = append(ans, q.questionToDomain(qu))
	}
	return ans, nil
}

func (q *questionRepository) questionToDomain(que dao.Question) domain.Question {
	return domain.Question{
		ID:      que.ID,
		UID:     que.UID,
		Title:   que.Title,
		Labels:  que.Labels,
		Content: que.Content,
		Status:  que.Status,
		Answer: domain.Answer{
			Analysis:     q.ansToDomain(que.Answer.Analysis),
			Basic:        q.ansToDomain(que.Answer.Basic),
			Intermediate: q.ansToDomain(que.Answer.Intermediate),
			Advanced:     q.ansToDomain(que.Answer.Advanced),
		},
	}
}

func (*questionRepository) ansToDomain(ans dao.AnswerElement) domain.AnswerElement {
	return domain.AnswerElement{
		ID:        ans.ID,
		Content:   ans.Content,
		Keywords:  ans.Keywords,
		Shorthand: ans.Shorthand,
		Highlight: ans.Highlight,
		Guidance:  ans.Guidance,
	}
}
