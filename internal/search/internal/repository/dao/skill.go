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
	SkillIndexName  = "skill_index"
	skillNameBoost  = 30
	skillLabelBoost = 6
	skillDescBoost  = 2
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
	ID           int64      `json:"id"`
	Labels       []string   `json:"labels"`
	Name         string     `json:"name"`
	Desc         string     `json:"desc"`
	Basic        SkillLevel `json:"basic"`
	Intermediate SkillLevel `json:"intermediate"`
	Advanced     SkillLevel `json:"advanced"`
	Ctime        int64      `json:"ctime"`
	Utime        int64      `json:"utime"`
}

type skillElasticDAO struct {
	client *elastic.Client
	metas  map[string]Col
}

func NewSkillElasticDAO(client *elastic.Client) SkillDAO {
	return &skillElasticDAO{
		client: client,
		metas: map[string]Col{
			"name": {
				Name:  "name",
				Boost: skillNameBoost,
			},
			"labels": {
				Name:   "labels",
				Boost:  skillLabelBoost,
				IsTerm: true,
			},
			"desc": {
				Name:  "desc",
				Boost: skillDescBoost,
			},
			"basic.desc": {
				Name: "basic.desc",
			},
			"intermediate.desc": {
				Name: "intermediate.desc",
			},
			"advanced.desc": {
				Name: "advanced.desc",
			},
		},
	}

}

func (s *skillElasticDAO) SearchSkill(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]Skill, error) {

	query := elastic.NewBoolQuery().Should(buildCols(s.metas, queryMetas)...)
	resp, err := s.client.Search(SkillIndexName).
		From(offset).
		Size(limit).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]Skill, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var ele Skill
		err = json.Unmarshal(hit.Source, &ele)
		if err != nil {
			return nil, err
		}
		res = append(res, ele)
	}
	return res, nil
}
