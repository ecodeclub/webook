package dao

import (
	"context"
	"encoding/json"

	"github.com/olivere/elastic/v7"
)

const (
	SkillIndexName  = "skill_index"
	skillNameBoost  = 15
	skillLabelBoost = 3
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
}

func NewSkillElasticDAO(client *elastic.Client) SkillDAO {
	return &skillElasticDAO{
		client: client,
	}
}

func (s *skillElasticDAO) SearchSkill(ctx context.Context, keywords string) ([]Skill, error) {

	query :=
		elastic.NewBoolQuery().Should(
			elastic.NewMatchQuery("name", keywords).Boost(skillNameBoost),
			elastic.NewMatchQuery("labels", keywords).Boost(skillLabelBoost),
			elastic.NewMatchQuery("desc", keywords).Boost(skillDescBoost),
			elastic.NewBoolQuery().Should(
				elastic.NewMatchQuery("basic.desc", keywords),
				elastic.NewMatchQuery("intermediate.desc", keywords),
				elastic.NewMatchQuery("advanced.desc", keywords),
			))

	resp, err := s.client.Search(SkillIndexName).Size(defaultSize).Query(query).Do(ctx)
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
