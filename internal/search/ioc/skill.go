package ioc

import (
	"github.com/ecodeclub/webook/internal/search/internal/repository/dao"
	"github.com/olivere/elastic/v7"
)

const (
	skillNameBoost  = 40
	skillLabelBoost = 6
	skillDescBoost  = 2
)

func InitSkillDAO(client *elastic.Client) dao.SkillDAO {
	metas := map[string]dao.FieldConfig{
		"name": {
			Name:  "name",
			Boost: skillNameBoost,
		},
		"labels": {
			Name:   "labels",
			Boost:  skillLabelBoost,
			IsTerm: true,
		},
		"desc": {
			Name:            "desc",
			Boost:           skillDescBoost,
			HighLightConfig: dao.DefaultHighlightConfig,
		},
		"basic.desc": {
			Name:            "basic.desc",
			HighLightConfig: dao.DefaultHighlightConfig,
		},
		"intermediate.desc": {
			Name:            "intermediate.desc",
			HighLightConfig: dao.DefaultHighlightConfig,
		},
		"advanced.desc": {
			Name:            "advanced.desc",
			HighLightConfig: dao.DefaultHighlightConfig,
		},
	}
	return dao.NewSkillDAO(client, metas)
}

func InitAdminSkillDAO(client *elastic.Client) dao.SkillDAO {
	metas := map[string]dao.FieldConfig{
		"name": {
			Name:  "name",
			Boost: skillNameBoost,
		},
		"labels": {
			Name:   "labels",
			Boost:  skillLabelBoost,
			IsTerm: true,
		},
		"desc": {
			Name:  "desc",
			Boost: skillDescBoost,
		},
		"basic.desc": {
			Name: "basic.desc",
		},
		"intermediate.desc": {
			Name: "intermediate.desc",
		},
		"advanced.desc": {
			Name: "advanced.desc",
		},
	}
	return dao.NewSkillDAO(client, metas)
}
