package dao

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/elastic/go-elasticsearch/v9"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
	"github.com/elastic/go-elasticsearch/v9/typedapi/types"
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

type searchData interface {
	SetEsHighLights(map[string][]string)
}

type searchClient[T searchData] struct {
	client     *elasticsearch.TypedClient
	index      string
	colsConfig map[string]FieldConfig
}

func (s searchClient[T]) build(cols map[string]FieldConfig,
	queryMetas []domain.QueryMeta, offset, limit int) map[string]any {
	queryQueries := s.buildQuery(cols, queryMetas)
	highLightCfg := s.buildHighLights(cols, queryMetas)
	var boolQuery *types.BoolQuery
	if len(queryQueries) > 0 {
		// 内层 BoolQuery 用于 Should 条件
		innerBool := types.NewBoolQuery()
		innerBool.Should = queryQueries
		// 外层 BoolQuery 用于 Must 条件
		boolQuery = types.NewBoolQuery()
		boolQuery.Must = []types.Query{
			{
				Bool: innerBool,
			},
		}
	}
	searchReq := map[string]any{
		"query": map[string]any{
			"bool": boolQuery,
		},
		"from": offset,
		"size": limit,
	}
	if highLightCfg != nil {
		highlightMap := make(map[string]any)
		if len(highLightCfg.PreTags) > 0 {
			highlightMap["pre_tags"] = highLightCfg.PreTags
		}
		if len(highLightCfg.PostTags) > 0 {
			highlightMap["post_tags"] = highLightCfg.PostTags
		}
		if len(highLightCfg.Fields) > 0 {
			fieldsMap := make(map[string]any)
			for _, fieldMap := range highLightCfg.Fields {
				for fieldName, fieldConfig := range fieldMap {
					fieldCfg := make(map[string]any)
					if fieldConfig.FragmentSize != nil {
						fieldCfg["fragment_size"] = *fieldConfig.FragmentSize
					}
					if fieldConfig.NumberOfFragments != nil {
						fieldCfg["number_of_fragments"] = *fieldConfig.NumberOfFragments
					}
					fieldsMap[fieldName] = fieldCfg
				}
			}
			highlightMap["fields"] = fieldsMap
		}
		searchReq["highlight"] = highlightMap
	}
	return searchReq
}

func (s searchClient[T]) getSearchCol(cols map[string]FieldConfig, queryMetas []domain.QueryMeta) map[string][]string {
	colMap := make(map[string][]string, len(cols))
	for _, meta := range queryMetas {
		if meta.IsAll {
			for _, col := range cols {
				colMap = s.setCol(colMap, col.Name, meta.Keyword)
			}
		} else {
			colMap = s.setCol(colMap, meta.Col, meta.Keyword)
		}
	}
	return colMap
}

func (s searchClient[T]) buildQuery(cols map[string]FieldConfig, queryMetas []domain.QueryMeta) []types.Query {
	colMap := s.getSearchCol(cols, queryMetas)
	queries := make([]types.Query, 0, len(colMap))
	for colname, keyword := range colMap {
		col := cols[colname]
		var query types.Query
		if col.IsTerm {
			// 构建 TermsQuery
			termsQuery := types.NewTermsQuery()
			// TermsQueryField 可以是 []FieldValue，而 FieldValue 可以是 string
			termsQuery.TermsQuery = map[string]types.TermsQueryField{
				colname: keyword, // 直接将 []string 转换为 TermsQueryField
			}
			if col.Boost > 0 {
				boost := float32(col.Boost)
				termsQuery.Boost = &boost
			}
			query = types.Query{
				Terms: termsQuery,
			}
		} else {
			// 构建 MatchQuery
			matchQuery := &types.MatchQuery{
				Query: strings.Join(keyword, " "),
			}
			if col.Boost > 0 {
				boost := float32(col.Boost)
				matchQuery.Boost = &boost
			}
			query = types.Query{
				Match: map[string]types.MatchQuery{
					colname: *matchQuery,
				},
			}
		}
		queries = append(queries, query)
	}
	return queries
}

func (s searchClient[T]) buildHighLights(cols map[string]FieldConfig, queryMetas []domain.QueryMeta) *types.Highlight {
	colMap := s.getSearchCol(cols, queryMetas)
	fields := make([]map[string]types.HighlightField, 0, len(cols))

	// 收集所有字段的 PreTags 和 PostTags（使用第一个字段的配置）
	var preTags, postTags []string
	hasHighlight := false

	for name, colField := range cols {
		if _, ok := colMap[name]; ok && colField.HighLightConfig.Status {
			hasHighlight = true
			field := s.buildHighLightConfig(colField)
			fields = append(fields, map[string]types.HighlightField{
				name: field,
			})
			// 使用第一个字段的 PreTags/PostTags 配置（通常所有字段使用相同的标签）
			if len(preTags) == 0 && len(colField.HighLightConfig.PreTag) > 0 {
				preTags = colField.HighLightConfig.PreTag
			}
			if len(postTags) == 0 && len(colField.HighLightConfig.PostTag) > 0 {
				postTags = colField.HighLightConfig.PostTag
			}
		}
	}

	if !hasHighlight {
		return nil
	}

	highlight := &types.Highlight{
		Fields: fields,
	}
	if len(preTags) > 0 {
		highlight.PreTags = preTags
	}
	if len(postTags) > 0 {
		highlight.PostTags = postTags
	}
	return highlight
}

func (s searchClient[T]) buildHighLightConfig(conf FieldConfig) types.HighlightField {
	field := types.HighlightField{}
	if conf.HighLightConfig.FragmentSize > 0 {
		field.FragmentSize = &conf.HighLightConfig.FragmentSize
	}
	if conf.HighLightConfig.FragmentsNumber > 0 {
		field.NumberOfFragments = &conf.HighLightConfig.FragmentsNumber
	}
	return field
}

func (s searchClient[T]) setCol(colMap map[string][]string, col, keyword string) map[string][]string {
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

func (s searchClient[T]) getSearchRes(
	ctx context.Context,
	queryMetas []domain.QueryMeta,
	offset, limit int) ([]T, error) {
	searchReq := s.build(s.colsConfig, queryMetas, offset, limit)
	// 执行搜索 - 使用 Raw 方法传入 JSON
	searchBytes, err := json.Marshal(searchReq)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Search().
		Index(s.index).
		Raw(bytes.NewReader(searchBytes)).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	// 解析结果
	res := make([]T, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var ele T
		if len(hit.Source_) > 0 {
			err = json.Unmarshal(hit.Source_, &ele)
			if err != nil {
				return nil, err
			}
		}
		ele.SetEsHighLights(getEsHighLights(hit.Highlight))
		res = append(res, ele)
	}
	return res, nil
}
