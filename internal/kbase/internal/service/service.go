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

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ecodeclub/ekit/retry"
	"github.com/ecodeclub/webook/internal/kbase/internal/domain"
)

// Service 知识库管理服务
//
//go:generate mockgen -source=./service.go -destination=../../mocks/service.mock.go -package=kbasemocks -typed=true Service
type Service interface {
	// BulkUpsert 批量插入或更新文档
	// 如果有失败则返回 error，调用方应重试整个批次
	BulkUpsert(ctx context.Context, indexName string, docs []domain.Document) error

	// BulkDelete 批量删除文档
	// 如果有失败则返回 error，调用方应重试整个批次
	BulkDelete(ctx context.Context, indexName string, docIDs []string) error
}

// 错误类型定义
var (
	// ErrClientError 客户端错误（4xx），不应重试
	ErrClientError = errors.New("客户端错误")
	// ErrServerError 服务端错误（5xx），应该重试
	ErrServerError = errors.New("服务端错误")
	// ErrNetworkError 网络错误，应该重试
	ErrNetworkError = errors.New("网络错误")
)

type ESErrorResponse struct {
	Error struct {
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error"`
	Status int `json:"status,omitempty"`
}

type ESBulkResponse struct {
	Took   int                           `json:"took"`
	Errors bool                          `json:"errors"`
	Items  []map[string]ESBulkItemResult `json:"items"`
}

type ESBulkItemResult struct {
	Index   string       `json:"_index"`
	ID      string       `json:"_id"`
	Version int          `json:"_version,omitempty"`
	Result  string       `json:"result,omitempty"` // created, updated, deleted, not_found
	Status  int          `json:"status"`
	Error   *ESItemError `json:"error,omitempty"`
}

type ESItemError struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// HTTPClient HTTP 客户端接口，用于执行 HTTP 请求
// 便于测试时 mock
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPKBaseService 基于 HTTP 的知识库服务实现
// 实现了 Service 接口
type HTTPKBaseService struct {
	baseURL     string
	client      HTTPClient
	batchSize   int           // 批次大小
	interval    time.Duration // 初始重试间隔
	maxInterval time.Duration // 最大重试间隔
	maxRetries  int           // 最大重试次数
}

// 编译时验证 HTTPKBaseService 实现了 Service 接口
var _ Service = (*HTTPKBaseService)(nil)

// NewHTTPKBaseService 创建 HTTP 知识库服务实例
// 参数:
//   - baseURL: kbase 服务的基础 URL，例如 "http://localhost:8082"
//   - client: HTTP 客户端，可以是 *http.Client 或 mock 实现
//   - batchSize: 批次大小，单次最多处理多少个文档
//   - interval: 初始重试间隔
//   - maxInterval: 最大重试间隔（用于指数退避）
//   - maxRetries: 最大重试次数
//
// 返回具体类型 *HTTPKBaseService 而不是接口，便于调用方进行扩展
func NewHTTPKBaseService(
	baseURL string,
	client HTTPClient,
	batchSize int,
	interval time.Duration,
	maxInterval time.Duration,
	maxRetries int,
) *HTTPKBaseService {
	return &HTTPKBaseService{
		baseURL:     baseURL,
		client:      client,
		batchSize:   batchSize,
		interval:    interval,
		maxInterval: maxInterval,
		maxRetries:  maxRetries,
	}
}

// BulkUpsert 批量插入或更新文档（带分批和重试）
func (s *HTTPKBaseService) BulkUpsert(ctx context.Context, indexName string, docs []domain.Document) error {
	return processBatches(ctx, docs, s.batchSize, func(ctx context.Context, batch []domain.Document) error {
		return s.bulkUpsertBatchWithRetry(ctx, indexName, batch)
	})
}

func processBatches[T any](
	ctx context.Context,
	items []T,
	batchSize int,
	processor func(ctx context.Context, batch []T) error,
) error {
	if len(items) == 0 {
		return nil
	}

	for i := 0; i < len(items); i += batchSize {
		// 检查 context 是否已取消
		if ctx.Err() != nil {
			return fmt.Errorf("context已取消: %w", ctx.Err())
		}

		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]
		if err := processor(ctx, batch); err != nil {
			return fmt.Errorf("批次 [%d:%d] 失败: %w", i, end, err)
		}
	}

	return nil
}

// bulkUpsertBatchWithRetry 带重试的单批处理
func (s *HTTPKBaseService) bulkUpsertBatchWithRetry(
	ctx context.Context,
	indexName string,
	docs []domain.Document,
) error {
	return s.doWithRetry(ctx, func() error {
		return s.bulkUpsertOnce(ctx, indexName, docs)
	})
}

func (s *HTTPKBaseService) doWithRetry(ctx context.Context, operation func() error) error {

	retryStrategy, err := retry.NewExponentialBackoffRetryStrategy(
		s.interval,
		s.maxInterval,
		int32(s.maxRetries),
	)

	if err != nil {
		return fmt.Errorf("创建重试策略失败: %w", err)
	}

	var lastErr error
	for {
		// 检查 context 是否已取消
		if ctx.Err() != nil {
			return fmt.Errorf("context已取消: %w", ctx.Err())
		}

		// 执行操作
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// 客户端错误不重试
		if errors.Is(err, ErrClientError) {
			return err
		}

		// 获取下一次重试的间隔
		next, ok := retryStrategy.Next()
		if !ok {
			return fmt.Errorf("超过最大重试次数，最后一次错误: %w", lastErr)
		}

		// 等待重试间隔
		select {
		case <-ctx.Done():
			return fmt.Errorf("context已取消: %w", ctx.Err())
		case <-time.After(next):
			// 继续重试
		}
	}
}

// bulkUpsertOnce 执行单次批量插入（不重试）
func (s *HTTPKBaseService) bulkUpsertOnce(
	ctx context.Context,
	indexName string,
	docs []domain.Document,
) error {
	reqBody := map[string]any{
		"index": indexName,
		"docs":  docs,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		// 序列化错误不应重试
		return fmt.Errorf("%w: 序列化请求失败: %v", ErrClientError, err)
	}

	// 使用 kbase 的新 API 端点
	url := fmt.Sprintf("%s/api/v1/bulk/upsert", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		// 创建请求失败不应重试
		return fmt.Errorf("%w: 创建请求失败: %v", ErrClientError, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		// 网络错误应该重试
		return fmt.Errorf("%w: 请求失败: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	// 根据状态码判断错误类型
	if resp.StatusCode != http.StatusOK {
		// 每次使用局部变量，避免并发问题
		var errorDetail struct {
			Detail string `json:"detail"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errorDetail)

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			// 4xx 客户端错误，不应重试
			if errorDetail.Detail != "" {
				return fmt.Errorf("%w: %s", ErrClientError, errorDetail.Detail)
			}
			return fmt.Errorf("%w: HTTP状态码=%d", ErrClientError, resp.StatusCode)
		}

		// 5xx 服务端错误，应该重试
		if errorDetail.Detail != "" {
			return fmt.Errorf("%w: %s", ErrServerError, errorDetail.Detail)
		}
		return fmt.Errorf("%w: HTTP状态码=%d", ErrServerError, resp.StatusCode)
	}

	// 解析 ES 响应
	var bulkResp ESBulkResponse
	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		// 解析失败可能是临时问题，可以重试
		return fmt.Errorf("%w: 解析响应失败: %v", ErrServerError, err)
	}

	// 检查是否有失败（业务语义：任何失败都返回 error）
	if bulkResp.Errors {
		failedCount := 0
		var firstError string

		for _, item := range bulkResp.Items {
			if indexResult, ok := item["index"]; ok {
				if indexResult.Status < 200 || indexResult.Status >= 300 {
					failedCount++
					if firstError == "" && indexResult.Error != nil {
						firstError = indexResult.Error.Reason
					}
				}
			}
		}

		if failedCount > 0 {
			// ES 批量操作失败，可能是临时问题，可以重试
			return fmt.Errorf("%w: 批量插入/更新失败: %d/%d 个文档失败，首个错误: %s",
				ErrServerError, failedCount, len(docs), firstError)
		}
	}

	return nil
}

// BulkDelete 批量删除文档（带分批和重试）
func (s *HTTPKBaseService) BulkDelete(ctx context.Context, indexName string, docIDs []string) error {
	return processBatches(ctx, docIDs, s.batchSize, func(ctx context.Context, batch []string) error {
		return s.bulkDeleteBatchWithRetry(ctx, indexName, batch)
	})
}

// bulkDeleteBatchWithRetry 带重试的单批处理
func (s *HTTPKBaseService) bulkDeleteBatchWithRetry(
	ctx context.Context,
	indexName string,
	docIDs []string,
) error {
	return s.doWithRetry(ctx, func() error {
		return s.bulkDeleteOnce(ctx, indexName, docIDs)
	})
}

// bulkDeleteOnce 执行单次批量删除
func (s *HTTPKBaseService) bulkDeleteOnce(
	ctx context.Context,
	indexName string,
	docIDs []string,
) error {
	reqBody := map[string]any{
		"index":   indexName,
		"doc_ids": docIDs,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("%w: 序列化请求失败: %v", ErrClientError, err)
	}

	// 使用 kbase 的新 API 端点（POST 方法）
	url := fmt.Sprintf("%s/api/v1/bulk/delete", s.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%w: 创建请求失败: %v", ErrClientError, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: 请求失败: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	// 根据状态码判断错误类型
	if resp.StatusCode != http.StatusOK {
		// 每次使用局部变量，避免并发问题
		var errorDetail struct {
			Detail string `json:"detail"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errorDetail)

		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			if errorDetail.Detail != "" {
				return fmt.Errorf("%w: %s", ErrClientError, errorDetail.Detail)
			}
			return fmt.Errorf("%w: HTTP状态码=%d", ErrClientError, resp.StatusCode)
		}

		if errorDetail.Detail != "" {
			return fmt.Errorf("%w: %s", ErrServerError, errorDetail.Detail)
		}
		return fmt.Errorf("%w: HTTP状态码=%d", ErrServerError, resp.StatusCode)
	}

	// 解析 ES 响应
	var bulkResp ESBulkResponse
	if err := json.NewDecoder(resp.Body).Decode(&bulkResp); err != nil {
		return fmt.Errorf("%w: 解析响应失败: %v", ErrServerError, err)
	}

	// 检查是否有失败（not_found 不算失败，幂等）
	if bulkResp.Errors {
		failedCount := 0
		var firstError string

		for _, item := range bulkResp.Items {
			if deleteResult, ok := item["delete"]; ok {
				// not_found 是幂等的，不算失败
				if deleteResult.Result == "not_found" {
					continue
				}

				if deleteResult.Status < 200 || deleteResult.Status >= 300 {
					failedCount++
					if firstError == "" && deleteResult.Error != nil {
						firstError = deleteResult.Error.Reason
					}
				}
			}
		}

		if failedCount > 0 {
			return fmt.Errorf("%w: 批量删除失败: %d/%d 个文档失败，首个错误: %s",
				ErrServerError, failedCount, len(docIDs), firstError)
		}
	}

	return nil
}
