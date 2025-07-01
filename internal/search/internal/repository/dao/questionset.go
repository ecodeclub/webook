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
	QuestionSetIndexName = "question_set_index"
)

type QuestionSet struct {
	Id  int64 `json:"id"`
	Uid int64 `json:"uid"`
	// 标题
	Title string `json:"title"`
	// 描述
	Description string `json:"description"`
	Biz         string `json:"biz"`
	BizID       int64  `json:"biz_id"`

	// 题集中引用的题目,
	Questions    []int64             `json:"questions"`
	Utime        int64               `json:"utime"`
	EsHighLights map[string][]string `json:"-"`
}
type questionSetElasticDAO struct {
	client  *elastic.Client
	builder searchBuilder
	metas   map[string]FieldConfig
}

func NewQuestionSetDAO(client *elastic.Client, metas map[string]FieldConfig) QuestionSetDAO {
	return &questionSetElasticDAO{
		client: client,
		metas:  metas,
	}
}

func (q *questionSetElasticDAO) SearchQuestionSet(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]QuestionSet, error) {
	cols, highlights := q.builder.build(q.metas, queryMetas)
	query := elastic.NewBoolQuery().Must(
		elastic.NewBoolQuery().Should(cols...))
	builder := q.client.Search(QuestionSetIndexName).
		From(offset).
		Size(limit).
		Query(query)
	if len(highlights) > 0 {
		builder = builder.Highlight(elastic.NewHighlight().Fields(highlights...))
	}
	resp, err := builder.Do(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]QuestionSet, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var (
			ele QuestionSet
		)
		err = json.Unmarshal(hit.Source, &ele)
		if err != nil {
			return nil, err
		}
		ele.EsHighLights = getEsHighLights(hit.Highlight)
		res = append(res, ele)
	}
	return res, nil

}
