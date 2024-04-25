package dao

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/olivere/elastic/v7"
)

const (
	QuestionIndexName    = "question_index"
	questionTitleBoost   = 11
	questionLabelBoost   = 10
	questionContentBoost = 2
)

type Question struct {
	ID      int64    `json:"id"`
	UID     int64    `json:"uid"`
	Title   string   `json:"title"`
	Labels  []string `json:"labels"`
	Content string   `json:"content"`
	Status  uint8    `json:"status"`
	Answer  Answer   `json:"answer"`
	Utime   int64    `json:"utime"`
}
type Answer struct {
	Analysis     AnswerElement `json:"analysis"`
	Basic        AnswerElement `json:"basic"`
	Intermediate AnswerElement `json:"intermediate"`
	Advanced     AnswerElement `json:"advanced"`
}

type AnswerElement struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Keywords  string `json:"keywords"`
	Shorthand string `json:"shorthand"`
	Highlight string `json:"highlight"`
	Guidance  string `json:"guidance"`
	Utime     int64  `json:"utime"`
}

type questionElasticDAO struct {
	client *elastic.Client
}

func NewQuestionDAO(client *elastic.Client) QuestionDAO {
	return &questionElasticDAO{
		client: client,
	}
}

func (q *questionElasticDAO) SearchQuestion(ctx context.Context, keywords []string) ([]Question, error) {
	queryString := strings.Join(keywords, " ")
	query := elastic.NewBoolQuery().Must(
		elastic.NewBoolQuery().Should(
			// 给予更高权重
			elastic.NewMatchQuery("title", queryString).Boost(questionTitleBoost),
			elastic.NewTermsQueryFromStrings("labels", keywords...).Boost(questionLabelBoost),
			elastic.NewMatchQuery("content", queryString).Boost(questionContentBoost),
			elastic.NewMatchQuery("answer.analysis.keywords", queryString),
			elastic.NewMatchQuery("answer.analysis.shorthand", queryString),
			elastic.NewMatchQuery("answer.analysis.highlight", queryString),
			elastic.NewMatchQuery("answer.analysis.guidance", queryString),
			elastic.NewMatchQuery("answer.basic.keywords", queryString),
			elastic.NewMatchQuery("answer.basic.shorthand", queryString),
			elastic.NewMatchQuery("answer.basic.highlight", queryString),
			elastic.NewMatchQuery("answer.basic.guidance", queryString),
			elastic.NewMatchQuery("answer.intermediate.keywords", queryString),
			elastic.NewMatchQuery("answer.intermediate.shorthand", queryString),
			elastic.NewMatchQuery("answer.intermediate.highlight", queryString),
			elastic.NewMatchQuery("answer.intermediate.guidance", queryString),
			elastic.NewMatchQuery("answer.advanced.keywords", queryString),
			elastic.NewMatchQuery("answer.advanced.shorthand", queryString),
			elastic.NewMatchQuery("answer.advanced.highlight", queryString),
			elastic.NewMatchQuery("answer.advanced.guidance", queryString)),
		elastic.NewTermQuery("status", 2))
	resp, err := q.client.Search(QuestionIndexName).Size(defaultSize).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]Question, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var ele Question
		err = json.Unmarshal(hit.Source, &ele)
		res = append(res, ele)
	}
	return res, nil
}

func (q *questionElasticDAO) InputQuestion(ctx context.Context, msg Question) error {
	_, err := q.client.Index().
		Index(CaseIndexName).
		Id(strconv.FormatInt(msg.ID, 10)).
		BodyJson(msg).Do(ctx)
	return err
}
