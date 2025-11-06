// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build e2e

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/ecodeclub/webook/internal/kbase/internal/domain"
	"github.com/ecodeclub/webook/internal/kbase/internal/service"
	"github.com/elastic/go-elasticsearch/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}

// ESHTTPClient 使用 go-elasticsearch 客户端实现的 HTTPClient
// 用于集成测试，将 kbase API 请求转换为 ES bulk API 调用
type ESHTTPClient struct {
	esClient *elasticsearch.Client
}

func NewESHTTPClient(esClient *elasticsearch.Client) *ESHTTPClient {
	return &ESHTTPClient{esClient: esClient}
}

func (c *ESHTTPClient) Do(req *http.Request) (*http.Response, error) {
	// 解析 kbase API 请求
	urlPath := req.URL.Path
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`{"detail": "读取请求体失败: %v"}`, err))),
		}, nil
	}

	// 根据 URL 路径处理不同的 API
	if strings.HasSuffix(urlPath, "/api/v1/bulk/upsert") {
		return c.handleBulkUpsert(req.Context(), bodyBytes)
	} else if strings.HasSuffix(urlPath, "/api/v1/bulk/delete") {
		return c.handleBulkDelete(req.Context(), bodyBytes)
	}

	return &http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader(`{"detail": "未知的API端点"}`)),
	}, nil
}

func (c *ESHTTPClient) handleBulkUpsert(ctx context.Context, bodyBytes []byte) (*http.Response, error) {
	// 解析 kbase API 请求体
	var reqBody struct {
		Index string            `json:"index"`
		Docs  []domain.Document `json:"docs"`
	}
	if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`{"detail": "解析请求失败: %v"}`, err))),
		}, nil
	}

	// 构建 ES bulk API 请求体
	var bulkBody strings.Builder
	for _, doc := range reqBody.Docs {
		action := map[string]any{
			"index": map[string]any{
				"_index": reqBody.Index,
				"_id":    doc.ID,
			},
		}
		actionBytes, _ := json.Marshal(action)
		bulkBody.Write(actionBytes)
		bulkBody.WriteString("\n")

		docBodyBytes, _ := json.Marshal(doc.Body)
		bulkBody.Write(docBodyBytes)
		bulkBody.WriteString("\n")
	}

	// 调用 ES bulk API
	resp, err := c.esClient.Bulk(strings.NewReader(bulkBody.String()), c.esClient.Bulk.WithContext(ctx))
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`{"detail": "ES请求失败: %v"}`, err))),
		}, nil
	}
	defer resp.Body.Close()

	// 读取 ES 响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`{"detail": "读取ES响应失败: %v"}`, err))),
		}, nil
	}

	// 返回 ES 响应（service.go 期望的格式）
	return &http.Response{
		StatusCode: resp.StatusCode,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     resp.Header,
	}, nil
}

func (c *ESHTTPClient) handleBulkDelete(ctx context.Context, bodyBytes []byte) (*http.Response, error) {
	// 解析 kbase API 请求体
	var reqBody struct {
		Index  string   `json:"index"`
		DocIDs []string `json:"doc_ids"`
	}
	if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`{"detail": "解析请求失败: %v"}`, err))),
		}, nil
	}

	// 构建 ES bulk API 请求体（删除操作）
	var bulkBody strings.Builder
	for _, docID := range reqBody.DocIDs {
		action := map[string]any{
			"delete": map[string]any{
				"_index": reqBody.Index,
				"_id":    docID,
			},
		}
		actionBytes, _ := json.Marshal(action)
		bulkBody.Write(actionBytes)
		bulkBody.WriteString("\n")
	}

	// 调用 ES bulk API
	resp, err := c.esClient.Bulk(strings.NewReader(bulkBody.String()), c.esClient.Bulk.WithContext(ctx))
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`{"detail": "ES请求失败: %v"}`, err))),
		}, nil
	}
	defer resp.Body.Close()

	// 读取 ES 响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(fmt.Sprintf(`{"detail": "读取ES响应失败: %v"}`, err))),
		}, nil
	}

	// 返回 ES 响应
	return &http.Response{
		StatusCode: resp.StatusCode,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     resp.Header,
	}, nil
}

type ServiceTestSuite struct {
	suite.Suite
	esClient *elasticsearch.Client
	svc      *service.HTTPKBaseService
	baseURL  string
}

func (s *ServiceTestSuite) SetupSuite() {
	// 初始化 ES 客户端
	cfg := elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
	}
	esClient, err := elasticsearch.NewClient(cfg)
	require.NoError(s.T(), err, "创建ES客户端失败")

	// 健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := esClient.Ping(esClient.Ping.WithContext(ctx))
	require.NoError(s.T(), err, "ES不可用（等待5秒超时），请先启动Elasticsearch服务")
	resp.Body.Close()

	s.esClient = esClient
	s.baseURL = "http://localhost:8082" // baseURL 用于构建请求路径，实际会被 ESHTTPClient 拦截

	// 创建基于 ES 的 HTTPClient
	httpClient := NewESHTTPClient(esClient)

	// 创建 HTTPKBaseService
	s.svc = service.NewHTTPKBaseService(
		s.baseURL,
		httpClient,
		100,                  // batchSize
		100*time.Millisecond, // interval
		6*time.Second,        // maxInterval
		3,                    // maxRetries
	)
}

func (s *ServiceTestSuite) TearDownTest() {
	// 清理测试索引
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 删除测试索引
	testIndices := []string{"test_index", "test_index_upsert", "test_index_delete"}
	for _, idx := range testIndices {
		resp, err := s.esClient.Indices.Delete([]string{idx}, s.esClient.Indices.Delete.WithContext(ctx))
		if err == nil && resp != nil {
			resp.Body.Close()
		}
	}
}

func (s *ServiceTestSuite) createTestIndex(ctx context.Context, indexName string) {
	// 删除已存在的索引
	s.esClient.Indices.Delete([]string{indexName}, s.esClient.Indices.Delete.WithContext(ctx))

	// 创建索引（使用简单的 mapping）
	resp, err := s.esClient.Indices.Create(indexName, s.esClient.Indices.Create.WithContext(ctx))
	require.NoError(s.T(), err)
	resp.Body.Close()
}

func (s *ServiceTestSuite) TestBulkUpsert() {
	t := s.T()
	testCases := []struct {
		name    string
		before  func(t *testing.T, indexName string)
		docs    []domain.Document
		wantErr bool
		after   func(t *testing.T, indexName string)
	}{
		{
			name: "空文档列表",
			before: func(t *testing.T, indexName string) {
				s.createTestIndex(t.Context(), indexName)
			},
			docs:    []domain.Document{},
			wantErr: false,
			after:   func(t *testing.T, indexName string) {},
		},
		{
			name: "单个文档",
			before: func(t *testing.T, indexName string) {
				s.createTestIndex(t.Context(), indexName)
			},
			docs: []domain.Document{
				{
					ID: "doc1",
					Body: map[string]any{
						"title":   "测试文档1",
						"content": "这是测试内容",
					},
				},
			},
			wantErr: false,
			after: func(t *testing.T, indexName string) {
				resp, err := s.esClient.Get(indexName, "doc1", s.esClient.Get.WithContext(t.Context()))
				require.NoError(t, err)
				defer resp.Body.Close()
				require.Equal(t, 200, resp.StatusCode, "文档应该存在")

				var result map[string]any
				err = json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				require.True(t, result["found"].(bool), "文档应该被找到")
				source := result["_source"].(map[string]any)
				assert.Equal(t, "测试文档1", source["title"])
				assert.Equal(t, "这是测试内容", source["content"])
			},
		},
		{
			name: "多个文档-小于批次大小",
			before: func(t *testing.T, indexName string) {
				s.createTestIndex(t.Context(), indexName)
			},
			docs: []domain.Document{
				{ID: "doc1", Body: map[string]any{"title": "文档1"}},
				{ID: "doc2", Body: map[string]any{"title": "文档2"}},
				{ID: "doc3", Body: map[string]any{"title": "文档3"}},
			},
			wantErr: false,
			after: func(t *testing.T, indexName string) {
				expectedDocs := map[string]map[string]any{
					"doc1": {"title": "文档1"},
					"doc2": {"title": "文档2"},
					"doc3": {"title": "文档3"},
				}
				for docID, expectedBody := range expectedDocs {
					resp, err := s.esClient.Get(indexName, docID, s.esClient.Get.WithContext(t.Context()))
					require.NoError(t, err)
					defer resp.Body.Close()
					require.Equal(t, 200, resp.StatusCode, "文档 %s 应该存在", docID)

					var result map[string]any
					err = json.NewDecoder(resp.Body).Decode(&result)
					require.NoError(t, err)
					require.True(t, result["found"].(bool), "文档 %s 应该被找到", docID)
					source := result["_source"].(map[string]any)
					for key, expectedValue := range expectedBody {
						assert.Equal(t, expectedValue, source[key], "文档 %s 的字段 %s 应该匹配", docID, key)
					}
				}
			},
		},
		{
			name: "多个文档-超过批次大小",
			before: func(t *testing.T, indexName string) {
				s.createTestIndex(t.Context(), indexName)
			},
			docs: func() []domain.Document {
				docs := make([]domain.Document, 0, 250)
				for i := 1; i <= 250; i++ {
					docs = append(docs, domain.Document{
						ID: fmt.Sprintf("doc%d", i),
						Body: map[string]any{
							"title": fmt.Sprintf("文档%d", i),
							"id":    i,
						},
					})
				}
				return docs
			}(),
			wantErr: false,
			after: func(t *testing.T, indexName string) {
				// 检查前几个和后几个文档
				for i := 1; i <= 5; i++ {
					docID := fmt.Sprintf("doc%d", i)
					resp, err := s.esClient.Get(indexName, docID, s.esClient.Get.WithContext(t.Context()))
					require.NoError(t, err)
					defer resp.Body.Close()
					require.Equal(t, 200, resp.StatusCode, "文档 %s 应该存在", docID)

					var result map[string]any
					err = json.NewDecoder(resp.Body).Decode(&result)
					require.NoError(t, err)
					require.True(t, result["found"].(bool), "文档 %s 应该被找到", docID)
					source := result["_source"].(map[string]any)
					assert.Equal(t, fmt.Sprintf("文档%d", i), source["title"])
					assert.Equal(t, float64(i), source["id"])
				}
				for i := 246; i <= 250; i++ {
					docID := fmt.Sprintf("doc%d", i)
					resp, err := s.esClient.Get(indexName, docID, s.esClient.Get.WithContext(t.Context()))
					require.NoError(t, err)
					defer resp.Body.Close()
					require.Equal(t, 200, resp.StatusCode, "文档 %s 应该存在", docID)

					var result map[string]any
					err = json.NewDecoder(resp.Body).Decode(&result)
					require.NoError(t, err)
					require.True(t, result["found"].(bool), "文档 %s 应该被找到", docID)
					source := result["_source"].(map[string]any)
					assert.Equal(t, fmt.Sprintf("文档%d", i), source["title"])
					assert.Equal(t, float64(i), source["id"])
				}
			},
		},
		{
			name: "更新已存在的文档",
			before: func(t *testing.T, indexName string) {
				s.createTestIndex(t.Context(), indexName)
				// 先插入一个文档
				doc := domain.Document{
					ID: "doc1",
					Body: map[string]any{
						"title":   "原始标题",
						"version": 1,
					},
				}
				err := s.svc.BulkUpsert(t.Context(), indexName, []domain.Document{doc})
				require.NoError(t, err)
			},
			docs: []domain.Document{
				{
					ID: "doc1",
					Body: map[string]any{
						"title":   "更新后的标题",
						"version": 2,
					},
				},
			},
			wantErr: false,
			after: func(t *testing.T, indexName string) {
				resp, err := s.esClient.Get(indexName, "doc1", s.esClient.Get.WithContext(t.Context()))
				require.NoError(t, err)
				defer resp.Body.Close()
				require.Equal(t, 200, resp.StatusCode, "文档应该存在")

				var result map[string]any
				err = json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				require.True(t, result["found"].(bool), "文档应该被找到")
				source := result["_source"].(map[string]any)
				assert.Equal(t, "更新后的标题", source["title"], "文档标题应该已更新")
				assert.Equal(t, float64(2), source["version"], "文档版本应该已更新")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			indexName := "test_index_upsert"
			tc.before(t, indexName)

			err := s.svc.BulkUpsert(t.Context(), indexName, tc.docs)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tc.after(t, indexName)
			}
		})
	}
}

func (s *ServiceTestSuite) TestBulkDelete() {
	t := s.T()
	testCases := []struct {
		name    string
		before  func(t *testing.T, indexName string) []string
		docIDs  []string
		wantErr bool
		after   func(t *testing.T, indexName string, docIDs []string)
	}{
		{
			name: "空ID列表",
			before: func(t *testing.T, indexName string) []string {
				s.createTestIndex(t.Context(), indexName)
				return []string{}
			},
			docIDs:  []string{},
			wantErr: false,
			after:   func(t *testing.T, indexName string, docIDs []string) {},
		},
		{
			name: "删除单个文档",
			before: func(t *testing.T, indexName string) []string {
				s.createTestIndex(t.Context(), indexName)
				docs := []domain.Document{
					{ID: "doc1", Body: map[string]any{"title": "文档1"}},
				}
				err := s.svc.BulkUpsert(t.Context(), indexName, docs)
				require.NoError(t, err)
				return []string{"doc1"}
			},
			docIDs:  []string{"doc1"},
			wantErr: false,
			after: func(t *testing.T, indexName string, docIDs []string) {
				resp, err := s.esClient.Get(indexName, "doc1", s.esClient.Get.WithContext(t.Context()))
				require.NoError(t, err)
				defer resp.Body.Close()
				require.Equal(t, 404, resp.StatusCode, "文档应该已被删除")

				var result map[string]any
				err = json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.False(t, result["found"].(bool), "文档应该不存在")
			},
		},
		{
			name: "删除多个文档",
			before: func(t *testing.T, indexName string) []string {
				s.createTestIndex(t.Context(), indexName)
				docs := []domain.Document{
					{ID: "doc1", Body: map[string]any{"title": "文档1"}},
					{ID: "doc2", Body: map[string]any{"title": "文档2"}},
					{ID: "doc3", Body: map[string]any{"title": "文档3"}},
				}
				err := s.svc.BulkUpsert(t.Context(), indexName, docs)
				require.NoError(t, err)
				return []string{"doc1", "doc2", "doc3"}
			},
			docIDs:  []string{"doc1", "doc2", "doc3"},
			wantErr: false,
			after: func(t *testing.T, indexName string, docIDs []string) {
				for _, docID := range docIDs {
					resp, err := s.esClient.Get(indexName, docID, s.esClient.Get.WithContext(t.Context()))
					require.NoError(t, err)
					defer resp.Body.Close()
					require.Equal(t, 404, resp.StatusCode, "文档 %s 应该已被删除", docID)

					var result map[string]any
					err = json.NewDecoder(resp.Body).Decode(&result)
					require.NoError(t, err)
					assert.False(t, result["found"].(bool), "文档 %s 应该不存在", docID)
				}
			},
		},
		{
			name: "删除不存在的文档-not_found应该幂等",
			before: func(t *testing.T, indexName string) []string {
				s.createTestIndex(t.Context(), indexName)
				return []string{"nonexistent"}
			},
			docIDs:  []string{"nonexistent"},
			wantErr: false, // not_found 应该幂等，不算失败
			after:   func(t *testing.T, indexName string, docIDs []string) {},
		},
		{
			name: "删除超过批次大小",
			before: func(t *testing.T, indexName string) []string {
				s.createTestIndex(t.Context(), indexName)
				docs := make([]domain.Document, 0, 250)
				docIDs := make([]string, 0, 250)
				for i := 1; i <= 250; i++ {
					docID := fmt.Sprintf("doc%d", i)
					docs = append(docs, domain.Document{
						ID:   docID,
						Body: map[string]any{"title": fmt.Sprintf("文档%d", i)},
					})
					docIDs = append(docIDs, docID)
				}
				err := s.svc.BulkUpsert(t.Context(), indexName, docs)
				require.NoError(t, err)
				return docIDs
			},
			docIDs: func() []string {
				docIDs := make([]string, 0, 250)
				for i := 1; i <= 250; i++ {
					docIDs = append(docIDs, fmt.Sprintf("doc%d", i))
				}
				return docIDs
			}(),
			wantErr: false,
			after: func(t *testing.T, indexName string, docIDs []string) {
				// 检查前几个和后几个文档
				for i := 1; i <= 5; i++ {
					docID := fmt.Sprintf("doc%d", i)
					resp, err := s.esClient.Get(indexName, docID, s.esClient.Get.WithContext(t.Context()))
					require.NoError(t, err)
					defer resp.Body.Close()
					require.Equal(t, 404, resp.StatusCode, "文档 %s 应该已被删除", docID)

					var result map[string]any
					err = json.NewDecoder(resp.Body).Decode(&result)
					require.NoError(t, err)
					assert.False(t, result["found"].(bool), "文档 %s 应该不存在", docID)
				}
				for i := 246; i <= 250; i++ {
					docID := fmt.Sprintf("doc%d", i)
					resp, err := s.esClient.Get(indexName, docID, s.esClient.Get.WithContext(t.Context()))
					require.NoError(t, err)
					defer resp.Body.Close()
					require.Equal(t, 404, resp.StatusCode, "文档 %s 应该已被删除", docID)

					var result map[string]any
					err = json.NewDecoder(resp.Body).Decode(&result)
					require.NoError(t, err)
					assert.False(t, result["found"].(bool), "文档 %s 应该不存在", docID)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			indexName := "test_index_delete"
			docIDs := tc.before(t, indexName)
			if len(tc.docIDs) > 0 {
				docIDs = tc.docIDs
			}

			err := s.svc.BulkDelete(t.Context(), indexName, docIDs)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tc.after(t, indexName, docIDs)
			}
		})
	}
}

func (s *ServiceTestSuite) TestContextCancel() {
	t := s.T()
	t.Run("BulkUpsert-Context取消", func(t *testing.T) {
		indexName := "test_index"
		s.createTestIndex(t.Context(), indexName)

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // 立即取消

		docs := []domain.Document{
			{ID: "doc1", Body: map[string]any{"title": "文档1"}},
		}
		err := s.svc.BulkUpsert(ctx, indexName, docs)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context已取消")
	})

	t.Run("BulkDelete-Context取消", func(t *testing.T) {
		indexName := "test_index"
		s.createTestIndex(t.Context(), indexName)

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // 立即取消

		err := s.svc.BulkDelete(ctx, indexName, []string{"doc1"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context已取消")
	})
}
