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

	"github.com/elastic/go-elasticsearch/v9"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
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

func (s *Skill) SetEsHighLights(highLights map[string][]string) {
	s.EsHighLights = highLights
}

type skillElasticDAO struct {
	client *searchClient[*Skill]
}

func NewSkillDAO(client *elasticsearch.TypedClient, metas map[string]FieldConfig) SkillDAO {
	return &skillElasticDAO{
		client: &searchClient[*Skill]{
			client:     client,
			index:      SkillIndexName,
			colsConfig: metas,
		},
	}
}
func (s *skillElasticDAO) SearchSkill(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]*Skill, error) {
	return s.client.getSearchRes(ctx, queryMetas, offset, limit)
}
