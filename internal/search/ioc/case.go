package ioc

import (
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
	"github.com/elastic/go-elasticsearch/v9"
)

const (
	caseTitleBoost    = 30
	caseLabelBoost    = 29
	caseKeywordsBoost = 3
	caseContentBoost  = 2
	caseGuidanceBoost = 1
)

func InitAdminCaseDAO(client *elasticsearch.TypedClient) dao.CaseDAO {
	metas := map[string]dao.FieldConfig{
		"title": {
			Name:  "title",
			Boost: caseTitleBoost,
		},
		"labels": {
			Name:   "labels",
			Boost:  caseLabelBoost,
			IsTerm: true,
		},
		"biz": {
			Name:   "biz",
			IsTerm: true,
		},
		"keywords": {
			Name:  "keywords",
			Boost: caseKeywordsBoost,
		},
		"shorthand": {
			Name:  "shorthand",
			Boost: caseKeywordsBoost,
		},
		"content": {
			Name:  "content",
			Boost: caseContentBoost,
		},
		"guidance": {
			Name:  "guidance",
			Boost: caseGuidanceBoost,
		},
	}
	return dao.NewCaseElasticDAO(client, metas, "case_index")
}

func InitCaseDAO(client *elasticsearch.TypedClient) dao.CaseDAO {
	metas := map[string]dao.FieldConfig{
		"title": {
			Name:  "title",
			Boost: caseTitleBoost,
		},
		"labels": {
			Name:   "labels",
			Boost:  caseLabelBoost,
			IsTerm: true,
		},
		"biz": {
			Name:   "biz",
			IsTerm: true,
		},
		"keywords": {
			Name:  "keywords",
			Boost: caseKeywordsBoost,
		},
		"shorthand": {
			Name:  "shorthand",
			Boost: caseKeywordsBoost,
		},
		"content": {
			Name:            "content",
			Boost:           caseContentBoost,
			HighLightConfig: dao.DefaultHighlightConfig,
		},
		"guidance": {
			Name:            "guidance",
			Boost:           caseGuidanceBoost,
			HighLightConfig: dao.DefaultHighlightConfig,
		},
	}
	return dao.NewCaseElasticDAO(client, metas, "pub_case_index")
}
