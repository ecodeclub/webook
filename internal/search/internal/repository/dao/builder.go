package dao

import (
	"strings"

	"github.com/ecodeclub/ekit/slice"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/olivere/elastic/v7"
)

// 用于构建查询
type FieldConfig struct {
	// 列名
	Name string
	// 权重
	Boost int
	// 是否是精确匹配
	IsTerm bool
}

type HighLightConfig struct {

}

func buildCols(cols map[string]FieldConfig, queryMetas []domain.QueryMeta) []elastic.Query {
	colMap := make(map[string][]string, len(cols))
	for _, meta := range queryMetas {
		if meta.IsAll {
			for _, col := range cols {
				colMap = setCol(colMap, col.Name, meta.Keyword)
			}
		} else {
			colMap = setCol(colMap, meta.Col, meta.Keyword)
		}
	}
	queries := make([]elastic.Query, 0, len(colMap))
	for colname, keyword := range colMap {
		col := cols[colname]
		var query elastic.Query
		if col.IsTerm {
			termVals := slice.Map(keyword, func(idx int, src string) any {
				return src
			})
			query = elastic.NewTermsQuery(colname, termVals...).Boost(float64(col.Boost))
		} else {
			query = elastic.NewMatchQuery(colname, strings.Join(keyword, " ")).Boost(float64(col.Boost))
		}
		queries = append(queries, query)
	}
	return queries
}

func setCol(colMap map[string][]string, col, keyword string) map[string][]string {
	ks, ok := colMap[col]
	if ok {
		ks = append(ks, keyword)
		colMap[col] = ks
	} else {
		colMap[col] = []string{
			keyword,
		}
	}
	return colMap
}
