package dao

import (
	"strings"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/olivere/elastic/v7"
)

// 用于构建查询
type Col struct {
	// 列名
	Name string
	// 权重
	Boost int
}

func buildCols(cols map[string]Col, queryMetas []domain.QueryMeta) []elastic.Query {
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
		query := elastic.NewMatchQuery(colname, strings.Join(keyword, " "))
		if col.Boost != 0 {
			query = query.Boost(float64(col.Boost))
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
