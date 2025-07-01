package dao

import "github.com/olivere/elastic/v7"

func getEsHighLights(field elastic.SearchHitHighlight) map[string][]string {
	highlights := make(map[string][]string)
	if field != nil {
		highlights = field
	}
	return highlights
}
