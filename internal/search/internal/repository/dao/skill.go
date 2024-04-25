package dao

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ecodeclub/ekit/slice"
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

func (s *skillElasticDAO) SearchSkill(ctx context.Context, qids, cids []int64, keywords []string) ([]Skill, error) {
	queryString := strings.Join(keywords, " ")
	qidList := slice.Map(qids, func(idx int, src int64) any {
		return src
	})
	cidList := slice.Map(cids, func(idx int, src int64) any {
		return src
	})
	query :=
		elastic.NewBoolQuery().Should(
			elastic.NewMatchQuery("name", queryString).Boost(skillNameBoost),
			elastic.NewTermsQueryFromStrings("labels", keywords...).Boost(skillLabelBoost),
			elastic.NewMatchQuery("desc", queryString).Boost(skillDescBoost),
			elastic.NewBoolQuery().Should(
				elastic.NewMatchQuery("basic.desc", queryString),
				elastic.NewTermsQuery("basic.cases", cidList...),
				elastic.NewTermsQuery("basic.questions", qidList...),
				elastic.NewMatchQuery("intermediate.desc", queryString),
				elastic.NewTermsQuery("intermediate.cases", cidList...),
				elastic.NewTermsQuery("intermediate.questions", qidList...),
				elastic.NewMatchQuery("advanced.desc", queryString),
				elastic.NewTermsQuery("advanced.cases", cidList...),
				elastic.NewTermsQuery("advanced.questions", qidList...)),
		)

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

func (s *skillElasticDAO) InputSkill(ctx context.Context, msg Skill) error {
	_, err := s.client.Index().
		Index(CaseIndexName).
		Id(strconv.FormatInt(msg.ID, 10)).
		BodyJson(msg).Do(ctx)
	return err
}
