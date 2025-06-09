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
	"encoding/json"

	"github.com/ecodeclub/webook/internal/search/internal/domain"

	"github.com/olivere/elastic/v7"
)

const (
	QuestionIndexName    = "question_index"
	questionTitleBoost   = 11
	questionLabelBoost   = 10
	questionContentBoost = 2
)

type Question struct {
	ID      int64    `json:"id"`
	UID     int64    `json:"uid"`
	Title   string   `json:"title"`
	Labels  []string `json:"labels"`
	Content string   `json:"content"`
	Status  uint8    `json:"status"`
	Answer  Answer   `json:"answer"`
	Utime   int64    `json:"utime"`
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
	client *elastic.Client
	metas  map[string]FieldConfig
}

func NewQuestionDAO(client *elastic.Client) QuestionDAO {
	return &questionElasticDAO{
		client: client,
		metas: map[string]FieldConfig{
			"title": {
				Name:  "title",
				Boost: questionTitleBoost,
			},
			"labels": {
				Name:   "labels",
				Boost:  questionLabelBoost,
				IsTerm: true,
			},
			"content": {
				Name:  "content",
				Boost: questionContentBoost,
			},
			"answer.analysis.keywords": {
				Name: "answer.analysis.keywords",
			},
			"answer.analysis.shorthand": {
				Name: "answer.analysis.shorthand",
			},
			"answer.analysis.highlight": {
				Name: "answer.analysis.highlight",
			},
			"answer.analysis.guidance": {
				Name: "answer.analysis.guidance",
			},
			"answer.basic.keywords": {
				Name: "answer.basic.keywords",
			},
			"answer.basic.shorthand": {
				Name: "answer.basic.shorthand",
			},
			"answer.basic.highlight": {
				Name: "answer.basic.highlight",
			},
			"answer.basic.guidance": {
				Name: "answer.basic.guidance",
			},
			"answer.intermediate.keywords": {
				Name: "answer.intermediate.keywords",
			},
			"answer.intermediate.shorthand": {
				Name: "answer.intermediate.shorthand",
			},
			"answer.intermediate.highlight": {
				Name: "answer.intermediate.highlight",
			},
			"answer.intermediate.guidance": {
				Name: "answer.intermediate.guidance",
			},
			"answer.advanced.keywords": {
				Name: "answer.advanced.keywords",
			},
			"answer.advanced.shorthand": {
				Name: "answer.advanced.shorthand",
			},
			"answer.advanced.highlight": {
				Name: "answer.advanced.highlight",
			},
			"answer.advanced.guidance": {
				Name: "answer.advanced.guidance",
			},
		},
	}
}

func (q *questionElasticDAO) SearchQuestion(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]Question, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewBoolQuery().Should(buildCols(q.metas, queryMetas)...),
		elastic.NewTermQuery("status", 2))
	resp, err := q.client.Search(QuestionIndexName).
		From(offset).
		Size(limit).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]Question, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var ele Question
		err = json.Unmarshal(hit.Source, &ele)
		if err != nil {
			return nil, err
		}
		res = append(res, ele)
	}
	return res, nil
}
