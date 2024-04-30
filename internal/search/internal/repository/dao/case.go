package dao

import (
	"context"
	"encoding/json"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/olivere/elastic/v7"
)

const CaseIndexName = "case_index"

// todo 添加分词器
type Case struct {
	Id        int64    `json:"id"`
	Uid       int64    `json:"uid"`
	Labels    []string `json:"labels"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	CodeRepo  string   `json:"code_repo"`
	Keywords  string   `json:"keywords"`
	Shorthand string   `json:"shorthand"`
	Highlight string   `json:"highlight"`
	Guidance  string   `json:"guidance"`
	Status    uint8    `json:"status"`
	Ctime     int64    `json:"ctime"`
	Utime     int64    `json:"utime"`
}
type CaseElasticDAO struct {
	client *elastic.Client
}

const (
	caseTitleBoost    = 30
	caseLabelBoost    = 29
	caseKeywordsBoost = 3
	caseContentBoost  = 2
	caseGuidanceBoost = 1
)

func (c *CaseElasticDAO) SearchCase(ctx context.Context, keywords string) ([]Case, error) {
	query := elastic.NewBoolQuery().Must(
		elastic.NewBoolQuery().Should(
			elastic.NewMatchQuery("title", keywords).Boost(caseTitleBoost),
			elastic.NewMatchQuery("labels", keywords).Boost(caseLabelBoost),
			elastic.NewMatchQuery("keywords", keywords).Boost(caseKeywordsBoost),
			elastic.NewMatchQuery("shorthand", keywords).Boost(caseKeywordsBoost),
			elastic.NewMatchQuery("content", keywords).Boost(caseContentBoost),
			elastic.NewMatchQuery("guidance", keywords).Boost(caseGuidanceBoost)),
		elastic.NewTermQuery("status", domain.PublishedStatus))
	resp, err := c.client.Search(CaseIndexName).Size(defaultSize).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]Case, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var ele Case
		err = json.Unmarshal(hit.Source, &ele)
		if err != nil {
			return nil, err
		}
		res = append(res, ele)
	}
	return res, nil
}

func NewCaseElasticDAO(client *elastic.Client) *CaseElasticDAO {
	return &CaseElasticDAO{
		client: client,
	}
}
