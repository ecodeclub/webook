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

	"github.com/ecodeclub/webook/internal/search/internal/domain"

	"github.com/elastic/go-elasticsearch/v9"
)

const (
	PubQuestionIndexName = "pub_question_index"
	QuestionIndexName    = "question_index"
)

type Question struct {
	ID           int64               `json:"id"`
	UID          int64               `json:"uid"`
	Biz          string              `json:"biz"`
	BizID        int64               `json:"biz_id"`
	Title        string              `json:"title"`
	Labels       []string            `json:"labels"`
	Content      string              `json:"content"`
	Status       uint8               `json:"status"`
	Answer       Answer              `json:"answer"`
	Utime        int64               `json:"utime"`
	EsHighLights map[string][]string `json:"-"`
}

func (q *Question) SetEsHighLights(highLights map[string][]string) {
	q.EsHighLights = highLights
}

type Answer struct {
	Analysis     AnswerElement `json:"analysis"`
	Basic        AnswerElement `json:"basic"`
	Intermediate AnswerElement `json:"intermediate"`
	Advanced     AnswerElement `json:"advanced"`
}

type AnswerElement struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Keywords  string `json:"keywords"`
	Shorthand string `json:"shorthand"`
	Highlight string `json:"highlight"`
	Guidance  string `json:"guidance"`
}

type questionElasticDAO struct {
	client *searchClient[*Question]
}

func NewQuestionElasticDAO(esClient *elasticsearch.TypedClient, index string, metas map[string]FieldConfig) QuestionDAO {
	return &questionElasticDAO{
		client: &searchClient[*Question]{
			client:     esClient,
			index:      index,
			colsConfig: metas,
		},
	}
}

func (q *questionElasticDAO) SearchQuestion(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]*Question, error) {
	return q.client.getSearchRes(ctx, queryMetas, offset, limit)
}
