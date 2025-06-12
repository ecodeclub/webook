package dao

import (
	"strings"

	"github.com/ecodeclub/ekit/slice"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/olivere/elastic/v7"
)

const (
	defaultPreTag         = "<strong>"
	defaultPostTag        = "</strong>"
	defaultFragmentSize   = 100
	defaultFragmentNumber = 5
)

var DefaultHighlightConfig = HighLightConfig{
	Status: true,
	PreTag: []string{
		defaultPreTag,
	},
	PostTag: []string{
		defaultPostTag,
	},
	FragmentSize:    defaultFragmentSize,
	FragmentsNumber: defaultFragmentNumber,
}

// 用于构建查询
type FieldConfig struct {
	// 列名
	Name string
	// 权重
	Boost int
	// 是否是精确匹配
	IsTerm          bool
	HighLightConfig HighLightConfig
}

type HighLightConfig struct {
	Status            bool
	PreTag            []string // 高亮前缀
	PostTag           []string // 高亮后缀
	FragmentSize      int      // 单个高亮片段的最大字符长度
	FragmentsNumber   int      // 返回的高亮片段数量
	RequireFieldMatch bool     // 是否高亮查询命中字段
}

type searchBuilder struct {
}

func newSearchBuilder() searchBuilder {
	return searchBuilder{}
}

func (s searchBuilder) build(cols map[string]FieldConfig, queryMetas []domain.QueryMeta) ([]elastic.Query, []*elastic.HighlighterField) {
	return s.buildQuery(cols, queryMetas), s.buildHighLights(cols, queryMetas)
}

func (s searchBuilder) getSearchCol(cols map[string]FieldConfig, queryMetas []domain.QueryMeta) map[string][]string {
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
	return colMap
}

func (s searchBuilder) buildQuery(cols map[string]FieldConfig, queryMetas []domain.QueryMeta) []elastic.Query {
	colMap := s.getSearchCol(cols, queryMetas)
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

func (s searchBuilder) buildHighLights(cols map[string]FieldConfig, queryMetas []domain.QueryMeta) []*elastic.HighlighterField {
	colMap := s.getSearchCol(cols, queryMetas)
	fields := make([]*elastic.HighlighterField, 0, len(cols))
	for name, colField := range cols {
		if _, ok := colMap[name]; ok && colField.HighLightConfig.Status {
			fields = append(fields, buildHighLightConfig(name, colField))
		}
	}
	return fields
}

func buildHighLightConfig(name string, conf FieldConfig) *elastic.HighlighterField {
	field := elastic.NewHighlighterField(name)
	if len(conf.HighLightConfig.PostTag) > 0 && len(conf.HighLightConfig.PreTag) > 0 {
		field = field.PostTags(conf.HighLightConfig.PostTag...).PreTags(conf.HighLightConfig.PreTag...)
	}
	if conf.HighLightConfig.FragmentSize > 0 {
		field = field.FragmentSize(conf.HighLightConfig.FragmentSize)
	}
	if conf.HighLightConfig.FragmentsNumber > 0 {
		field = field.NumOfFragments(conf.HighLightConfig.FragmentsNumber)
	}
	return field
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
