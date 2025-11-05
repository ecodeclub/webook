package ioc

import (
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
	"github.com/elastic/go-elasticsearch/v9"
)

const (
	questionSetTitleBoost  = 10
	questionSetDescription = 2
)

func InitQuestionSetDAO(client *elasticsearch.TypedClient) dao.QuestionSetDAO {
	metas := map[string]dao.FieldConfig{
		"title": {
			Name:  "title",
			Boost: questionSetTitleBoost,
		},
		"biz": {
			Name:   "biz",
			IsTerm: true,
		},
		"description": {
			Name:            "description",
			Boost:           questionSetDescription,
			HighLightConfig: dao.DefaultHighlightConfig,
		},
	}
	return dao.NewQuestionSetDAO(client, metas)
}

func InitAdminQuestionSetDAO(client *elasticsearch.TypedClient) dao.QuestionSetDAO {
	metas := map[string]dao.FieldConfig{
		"title": {
			Name:  "title",
			Boost: questionSetTitleBoost,
		},
		"biz": {
			Name:   "biz",
			IsTerm: true,
		},
		"description": {
			Name:  "description",
			Boost: questionSetDescription,
		},
	}
	return dao.NewQuestionSetDAO(client, metas)
}
