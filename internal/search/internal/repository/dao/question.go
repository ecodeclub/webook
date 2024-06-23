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
}

func NewQuestionDAO(client *elastic.Client) QuestionDAO {
	return &questionElasticDAO{
		client: client,
	}
}

func (q *questionElasticDAO) SearchQuestion(ctx context.Context, offset, limit int, keywords string) ([]Question, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewBoolQuery().Should(
			// 给予更高权重
			elastic.NewMatchQuery("title", keywords).Boost(questionTitleBoost),
			elastic.NewMatchQuery("labels", keywords).Boost(questionLabelBoost),
			elastic.NewMatchQuery("content", keywords).Boost(questionContentBoost),
			elastic.NewMatchQuery("answer.analysis.keywords", keywords),
			elastic.NewMatchQuery("answer.analysis.shorthand", keywords),
			elastic.NewMatchQuery("answer.analysis.highlight", keywords),
			elastic.NewMatchQuery("answer.analysis.guidance", keywords),
			elastic.NewMatchQuery("answer.basic.keywords", keywords),
			elastic.NewMatchQuery("answer.basic.shorthand", keywords),
			elastic.NewMatchQuery("answer.basic.highlight", keywords),
			elastic.NewMatchQuery("answer.basic.guidance", keywords),
			elastic.NewMatchQuery("answer.intermediate.keywords", keywords),
			elastic.NewMatchQuery("answer.intermediate.shorthand", keywords),
			elastic.NewMatchQuery("answer.intermediate.highlight", keywords),
			elastic.NewMatchQuery("answer.intermediate.guidance", keywords),
			elastic.NewMatchQuery("answer.advanced.keywords", keywords),
			elastic.NewMatchQuery("answer.advanced.shorthand", keywords),
			elastic.NewMatchQuery("answer.advanced.highlight", keywords),
			elastic.NewMatchQuery("answer.advanced.guidance", keywords)),
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
