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

type skillRepo struct {
	skillDao dao.SkillDAO
}

func NewSKillRepo(skillDao dao.SkillDAO) SkillRepo {
	return &skillRepo{
		skillDao: skillDao,
	}
}

func (s *skillRepo) SearchSkill(ctx context.Context, offset, limit int, queryMetas []domain.QueryMeta) ([]domain.Skill, error) {
	skillList, err := s.skillDao.SearchSkill(ctx, offset, limit, queryMetas)
	if err != nil {
		return nil, err
	}
	ans := make([]domain.Skill, 0, len(skillList))
	for _, sk := range skillList {
		ans = append(ans, s.toSkillDomain(sk))
	}
	return ans, nil
}

func (sk *skillRepo) toSkillDomain(s *dao.Skill) domain.Skill {
	return domain.Skill{
		ID:     s.ID,
		Labels: s.Labels,
		Name:   s.Name,
		Desc: domain.EsVal{
			Val:           s.Desc,
			HighLightVals: s.EsHighLights["desc"],
		},
		Basic:        sk.toSkillLevelDomain(s.Basic, s.EsHighLights, "basic.desc"),
		Intermediate: sk.toSkillLevelDomain(s.Intermediate, s.EsHighLights, "intermediate.desc"),
		Advanced:     sk.toSkillLevelDomain(s.Advanced, s.EsHighLights, "advanced.desc"),
		Ctime:        time.UnixMilli(s.Ctime),
		Utime:        time.UnixMilli(s.Utime),
	}
}

func (s *skillRepo) toSkillLevelDomain(sk dao.SkillLevel, esHighlights map[string][]string, typ string) domain.SkillLevel {
	return domain.SkillLevel{
		ID:        sk.ID,
		Desc:      domain.EsVal{Val: sk.Desc, HighLightVals: esHighlights[typ]},
		Ctime:     time.UnixMilli(sk.Ctime),
		Utime:     time.UnixMilli(sk.Utime),
		Questions: sk.Questions,
		Cases:     sk.Cases,
	}
}
