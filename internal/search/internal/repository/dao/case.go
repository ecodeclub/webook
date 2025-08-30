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
	PubCaseIndexName = "pub_case_index"
	CaseIndexName    = "case_index"
)

// todo 添加分词器
type Case struct {
	Id           int64               `json:"id"`
	Uid          int64               `json:"uid"`
	Biz          string              `json:"biz"`
	BizID        int64               `json:"biz_id"`
	Labels       []string            `json:"labels"`
	Title        string              `json:"title"`
	Content      string              `json:"content"`
	GithubRepo   string              `json:"github_repo"`
	GiteeRepo    string              `json:"gitee_repo"`
	Keywords     string              `json:"keywords"`
	Shorthand    string              `json:"shorthand"`
	Highlight    string              `json:"highlight"`
	Guidance     string              `json:"guidance"`
	Status       uint8               `json:"status"`
	Ctime        int64               `json:"ctime"`
	Utime        int64               `json:"utime"`
	EsHighLights map[string][]string `json:"-"`
}

type CaseElasticDAO struct {
	client  *elastic.Client
	metas   map[string]FieldConfig
	builder searchBuilder
	index   string
}

func (c *CaseElasticDAO) SearchCase(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]Case, error) {
	cols, highlights := c.builder.build(c.metas, queryMetas)
	query := elastic.NewBoolQuery().Must(
		elastic.NewBoolQuery().Should(cols...))
	builder := c.client.Search(c.index).
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
	res := make([]Case, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var (
			ele Case
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

func NewCaseElasticDAO(client *elastic.Client, metas map[string]FieldConfig, index string) *CaseElasticDAO {
	return &CaseElasticDAO{
		client:  client,
		metas:   metas,
		index:   index,
		builder: newSearchBuilder(),
	}
}
