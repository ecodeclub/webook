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
	SkillIndexName = "skill_index"
)

type SkillLevel struct {
	ID        int64   `json:"id"`
	Desc      string  `json:"desc"`
	Ctime     int64   `json:"ctime"`
	Utime     int64   `json:"utime"`
	Questions []int64 `json:"questions"`
	Cases     []int64 `json:"cases"`
}

type Skill struct {
	ID           int64               `json:"id"`
	Labels       []string            `json:"labels"`
	Name         string              `json:"name"`
	Desc         string              `json:"desc"`
	Basic        SkillLevel          `json:"basic"`
	Intermediate SkillLevel          `json:"intermediate"`
	Advanced     SkillLevel          `json:"advanced"`
	EsHighLights map[string][]string `json:"-"`
	Ctime        int64               `json:"ctime"`
	Utime        int64               `json:"utime"`
}

type skillElasticDAO struct {
	client  *elastic.Client
	metas   map[string]FieldConfig
	builder searchBuilder
}

func NewSkillDAO(client *elastic.Client, metas map[string]FieldConfig) SkillDAO {
	return &skillElasticDAO{
		client: client,
		metas:  metas,
	}
}
func (s *skillElasticDAO) SearchSkill(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]Skill, error) {
	cols, highlights := s.builder.build(s.metas, queryMetas)
	query := elastic.NewBoolQuery().Must(
		elastic.NewBoolQuery().Should(cols...))
	builder := s.client.Search(SkillIndexName).
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
	res := make([]Skill, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var (
			ele Skill
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
