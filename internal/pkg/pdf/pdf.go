package pdf

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Converter PDF转换器接口
type Converter interface {
	// ConvertHTMLToPDF 将HTML内容转换为PDF
	ConvertHTMLToPDF(ctx context.Context, html string, opts ...Option) ([]byte, error)
}

// Options 保留为向后兼容（当前远程服务未支持自定义参数）
type Options struct {
	PaperWidthInch   float64
	PaperHeightInch  float64
	MarginTopInch    float64
	MarginBottomInch float64
	MarginLeftInch   float64
	MarginRightInch  float64
	Landscape        bool
	Title            string
}

// Option 配置选项函数类型（当前未被远程服务使用）
type Option func(*Options)

// RemotePDFConverter 通过远程HTTP服务进行PDF转换
type RemotePDFConverter struct {
	// Endpoint 完整的转换接口地址，例如: http://localhost:9999/pdf/convert
	Endpoint   string
	HTTPClient *http.Client
}

// NewRemotePDFConverter 创建远程转换器
func NewRemotePDFConverter(endpoint string) *RemotePDFConverter {
	return &RemotePDFConverter{
		Endpoint: endpoint,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ConvertHTMLToPDF 调用远程HTTP接口生成PDF
func (c *RemotePDFConverter) ConvertHTMLToPDF(ctx context.Context, html string, _ ...Option) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 编码为base64
	encoded := base64.StdEncoding.EncodeToString([]byte(html))

	// 组装请求体
	payload := struct {
		Data string `json:"data"`
	}{Data: encoded}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if readErr != nil {
			return nil, fmt.Errorf("remote error: status=%d", resp.StatusCode)
		}
		return nil, fmt.Errorf("remote error: status=%d, body=%s", resp.StatusCode, string(respBytes))
	}
	if readErr != nil {
		return nil, fmt.Errorf("read response failed: %w", readErr)
	}
	return respBytes, nil
}



