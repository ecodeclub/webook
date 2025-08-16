package ioc

import (
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
	"github.com/olivere/elastic/v7"
)

const (
	questionTitleBoost   = 11
	questionLabelBoost   = 10
	questionContentBoost = 2
)

func InitQuestionDAO(client *elastic.Client) dao.QuestionDAO {
	meta := map[string]dao.FieldConfig{
		"title": {
			Name:  "title",
			Boost: questionTitleBoost,
		},
		"labels": {
			Name:   "labels",
			Boost:  questionLabelBoost,
			IsTerm: true,
		},
		"biz": {
			Name:   "biz",
			IsTerm: true,
		},
		"content": {
			Name:            "content",
			Boost:           questionContentBoost,
			HighLightConfig: dao.DefaultHighlightConfig,
		},
		"answer.analysis.keywords": {
			Name: "answer.analysis.keywords",
		},
		"answer.analysis.shorthand": {
			Name: "answer.analysis.shorthand",
		},
		"answer.analysis.highlight": {
			Name: "answer.analysis.highlight",
		},
		"answer.analysis.guidance": {
			Name: "answer.analysis.guidance",
		},
		"answer.analysis.content": {
			Name:            "answer.analysis.content",
			HighLightConfig: dao.DefaultHighlightConfig,
		},
		"answer.basic.keywords": {
			Name: "answer.basic.keywords",
		},
		"answer.basic.shorthand": {
			Name: "answer.basic.shorthand",
		},
		"answer.basic.highlight": {
			Name: "answer.basic.highlight",
		},
		"answer.basic.guidance": {
			Name: "answer.basic.guidance",
		},
		"answer.basic.content": {
			Name:            "answer.basic.content",
			HighLightConfig: dao.DefaultHighlightConfig,
		},
		"answer.intermediate.keywords": {
			Name: "answer.intermediate.keywords",
		},
		"answer.intermediate.shorthand": {
			Name: "answer.intermediate.shorthand",
		},
		"answer.intermediate.highlight": {
			Name: "answer.intermediate.highlight",
		},
		"answer.intermediate.guidance": {
			Name: "answer.intermediate.guidance",
		},
		"answer.intermediate.content": {
			Name:            "answer.intermediate.content",
			HighLightConfig: dao.DefaultHighlightConfig,
		},
		"answer.advanced.keywords": {
			Name: "answer.advanced.keywords",
		},
		"answer.advanced.shorthand": {
			Name: "answer.advanced.shorthand",
		},
		"answer.advanced.highlight": {
			Name: "answer.advanced.highlight",
		},
		"answer.advanced.guidance": {
			Name: "answer.advanced.guidance",
		},
		"answer.advanced.content": {
			Name:            "answer.advanced.content",
			HighLightConfig: dao.DefaultHighlightConfig,
		},
	}
	return dao.NewQuestionElasticDAO(client, "pub_question_index", meta)
}

func InitAdminQuestionDAO(client *elastic.Client) dao.QuestionDAO {
	meta := map[string]dao.FieldConfig{
		"title": {
			Name:  "title",
			Boost: questionTitleBoost,
		},
		"labels": {
			Name:   "labels",
			Boost:  questionLabelBoost,
			IsTerm: true,
		},
		"biz": {
			Name:   "biz",
			IsTerm: true,
		},
		"content": {
			Name:  "content",
			Boost: questionContentBoost,
		},
		"answer.analysis.keywords": {
			Name: "answer.analysis.keywords",
		},
		"answer.analysis.shorthand": {
			Name: "answer.analysis.shorthand",
		},
		"answer.analysis.highlight": {
			Name: "answer.analysis.highlight",
		},
		"answer.analysis.content": {
			Name: "answer.analysis.content",
		},
		"answer.analysis.guidance": {
			Name: "answer.analysis.guidance",
		},
		"answer.basic.keywords": {
			Name: "answer.basic.keywords",
		},
		"answer.basic.shorthand": {
			Name: "answer.basic.shorthand",
		},
		"answer.basic.highlight": {
			Name: "answer.basic.highlight",
		},
		"answer.basic.content": {
			Name: "answer.analysis.content",
		},
		"answer.basic.guidance": {
			Name: "answer.basic.guidance",
		},
		"answer.intermediate.keywords": {
			Name: "answer.intermediate.keywords",
		},
		"answer.intermediate.shorthand": {
			Name: "answer.intermediate.shorthand",
		},
		"answer.intermediate.highlight": {
			Name: "answer.intermediate.highlight",
		},
		"answer.intermediate.content": {
			Name: "answer.intermediate.content",
		},
		"answer.intermediate.guidance": {
			Name: "answer.intermediate.guidance",
		},
		"answer.advanced.keywords": {
			Name: "answer.advanced.keywords",
		},
		"answer.advanced.shorthand": {
			Name: "answer.advanced.shorthand",
		},
		"answer.advanced.highlight": {
			Name: "answer.advanced.highlight",
		},
		"answer.advanced.content": {
			Name: "answer.advanced.content",
		},
		"answer.advanced.guidance": {
			Name: "answer.advanced.guidance",
		},
	}
	return dao.NewQuestionElasticDAO(client, "question_index", meta)
}
