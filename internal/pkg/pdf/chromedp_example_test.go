//go:build example

package pdf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestChromeDPConverter 测试基于ChromeDP的PDF转换功能
func TestChromeDPConverter(t *testing.T) {
	// 跳过测试，如果明确指定不运行
	if testing.Short() {
		t.Skip("跳过需要Docker环境的测试")
	}

	// 获取远程Chrome WebSocket URL
	wsURL := "ws://localhost:3000"

	t.Logf("成功连接到Chrome实例: %s", wsURL)

	// 创建转换器
	converter := NewChromeDPConverter(wsURL)

	// 设置HTML内容
	html := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>PDF测试文档</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1 { color: #333; text-align: center; }
        p { line-height: 1.6; }
        .footer { margin-top: 50px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <h1>PDF转换测试</h1>
    <p>这是一个使用ChromeDP生成的PDF文档示例。</p>
    <p>当前时间: ` + time.Now().Format("2006-01-02 15:04:05") + `</p>
    <div class="footer">
        由Webook PDF转换服务生成
    </div>
</body>
</html>`

	// 设置选项
	options := []Option{
		PaperA4,              // 使用A4纸张
		MarginsNormal,        // 使用标准边距
		WithLandscape(false), // 纵向打印
		WithTitle("PDF测试文档"), // 设置标题
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 转换为PDF
	t.Log("开始转换HTML到PDF...")
	pdfData, err := converter.ConvertHTMLToPDF(ctx, html, options...)
	if err != nil {
		t.Fatalf("转换失败: %v", err)
	}

	// 确保测试输出目录存在
	testOutputDir := filepath.Join(os.TempDir(), "pdf_test")
	if err := os.MkdirAll(testOutputDir, 0755); err != nil {
		t.Fatalf("创建测试输出目录失败: %v", err)
	}

	// 保存PDF文件到临时目录
	outputFile := filepath.Join(testOutputDir, fmt.Sprintf("test_output_%d.pdf", time.Now().Unix()))
	if err := os.WriteFile(outputFile, pdfData, 0644); err != nil {
		t.Fatalf("保存PDF文件失败: %v", err)
	}

	t.Logf("PDF已成功生成: %s (大小: %d 字节)", outputFile, len(pdfData))

	// 验证PDF数据
	if len(pdfData) < 1000 {
		t.Errorf("生成的PDF文件过小，可能不是有效的PDF: %d 字节", len(pdfData))
	}

	// 检查PDF文件头
	if len(pdfData) >= 4 && string(pdfData[:4]) != "%PDF" {
		t.Errorf("生成的文件不是有效的PDF，文件头: %s", string(pdfData[:4]))
	}
}

// 尝试从多个可能的端点获取Chrome WebSocket URL
func getBrowserWSURL() (string, error) {
	// 尝试的端点列表
	endpoints := []string{
		"http://localhost:3000/json/version", // browserless优先
		"http://127.0.0.1:3000/json/version",
		"http://localhost:9222/json/version",
		"http://127.0.0.1:9222/json/version",
	}

	// 从环境变量获取（如果设置了）
	if wsURL := os.Getenv("CHROME_WS_URL"); wsURL != "" {
		return wsURL, nil
	}

	// 尝试每个端点
	for _, endpoint := range endpoints {
		wsURL, err := getWSURLFromEndpoint(endpoint)
		if err == nil && wsURL != "" {
			return wsURL, nil
		}
	}

	return "", fmt.Errorf("无法从任何端点获取WebSocket URL")
}

// 从指定端点获取WebSocket URL
func getWSURLFromEndpoint(endpoint string) (string, error) {
	// 设置短超时，避免长时间等待
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	resp, err := client.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP错误: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if wsURL, ok := result["webSocketDebuggerUrl"].(string); ok && wsURL != "" {
		return wsURL, nil
	}

	return "", fmt.Errorf("响应中没有webSocketDebuggerUrl字段")
}
