package dao

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

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
	caseKeywordsBoost = 15
	caseContentBoost  = 10
	caseGuidanceBoost = 5
)

func (c *CaseElasticDAO) SearchCase(ctx context.Context, keywords []string) ([]Case, error) {
	queryString := strings.Join(keywords, " ")

	query := elastic.NewBoolQuery().Must(
		elastic.NewBoolQuery().Should(
			elastic.NewMatchQuery("title", queryString).Boost(caseTitleBoost),
			elastic.NewTermsQueryFromStrings("labels", queryString).Boost(caseLabelBoost),
			elastic.NewMatchQuery("keywords", queryString).Boost(caseKeywordsBoost),
			elastic.NewMatchQuery("shorthand", queryString).Boost(caseKeywordsBoost),
			elastic.NewMatchQuery("content", queryString).Boost(caseContentBoost),
			elastic.NewMatchQuery("guidance", queryString).Boost(caseGuidanceBoost)),
		elastic.NewTermQuery("status", domain.PublishedStatus))
	resp, err := c.client.Search(CaseIndexName).Size(defaultSize).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]Case, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var ele Case
		err = json.Unmarshal(hit.Source, &ele)
		res = append(res, ele)
	}
	return res, nil
}

func (c *CaseElasticDAO) InputCase(ctx context.Context, msg Case) error {
	_, err := c.client.Index().
		Index(CaseIndexName).
		Id(strconv.FormatInt(msg.Id, 10)).
		BodyJson(msg).Do(ctx)
	return err
}

func NewCaseElasticDAO(client *elastic.Client) *CaseElasticDAO {
	return &CaseElasticDAO{
		client: client,
	}
}
