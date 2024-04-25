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

func (q *questionSetElasticDAO) SearchQuestionSet(ctx context.Context, qids []int64, keywords []string) ([]QuestionSet, error) {
	queryString := strings.Join(keywords, " ")
	qidList := slice.Map(qids, func(idx int, src int64) any {
		return src
	})
	query := elastic.NewBoolQuery().Should(
		// 给予更高权重
		elastic.NewMatchQuery("title", queryString).Boost(questionSetTitleBoost),
		elastic.NewMatchQuery("description", queryString).Boost(questionSetDescription),
		elastic.NewTermsQuery("questions", qidList...),
	)
	resp, err := q.client.Search(QuestionSetIndexName).Size(defaultSize).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]QuestionSet, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var ele QuestionSet
		err = json.Unmarshal(hit.Source, &ele)
		res = append(res, ele)
	}
	return res, nil

}

func (q *questionSetElasticDAO) InputQuestionSet(ctx context.Context, msg QuestionSet) error {
	_, err := q.client.Index().
		Index(CaseIndexName).
		Id(strconv.FormatInt(msg.Id, 10)).
		BodyJson(msg).Do(ctx)
	return err
}
