package dao

import (
	"context"
	"encoding/json"

	"github.com/olivere/elastic/v7"
)

const (
	QuestionSetIndexName   = "question_set_index"
	questionSetTitleBoost  = 10
	questionSetDescription = 2
)

type QuestionSet struct {
	Id  int64 `json:"id"`
	Uid int64 `json:"uid"`
	// 标题
	Title string `json:"title"`
	// 描述
	Description string `json:"description"`

	// 题集中引用的题目,
	Questions []int64 `json:"questions"`
	Utime     int64   `json:"utime"`
}
type questionSetElasticDAO struct {
	client *elastic.Client
}

func NewQuestionSetDAO(client *elastic.Client) QuestionSetDAO {
	return &questionSetElasticDAO{
		client: client,
	}
}

func (q *questionSetElasticDAO) SearchQuestionSet(ctx context.Context, keywords string) ([]QuestionSet, error) {
	query := elastic.NewBoolQuery().Should(
		// 给予更高权重
		elastic.NewMatchQuery("title", keywords).Boost(questionSetTitleBoost),
		elastic.NewMatchQuery("description", keywords).Boost(questionSetDescription),
	)
	resp, err := q.client.Search(QuestionSetIndexName).Size(defaultSize).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]QuestionSet, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var ele QuestionSet
		err = json.Unmarshal(hit.Source, &ele)
		if err != nil {
			return nil, err
		}
		res = append(res, ele)
	}
	return res, nil

}
