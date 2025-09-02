package pdf

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// ChromeDPConverter 使用ChromeDP将HTML转换为PDF
type ChromeDPConverter struct {
	// 远程Chrome WebSocket URL
	RemoteWebSocketURL string
	// 默认超时时间
	DefaultTimeout time.Duration
	// 默认PDF选项
	DefaultOptions Options
}

// NewChromeDPConverter 创建一个基于ChromeDP的PDF转换器
func NewChromeDPConverter(remoteWebSocketURL string) *ChromeDPConverter {
	return &ChromeDPConverter{
		RemoteWebSocketURL: remoteWebSocketURL,
		DefaultTimeout:     60 * time.Second,
		DefaultOptions: Options{
			PaperWidthInch:   8.5,
			PaperHeightInch:  11,
			MarginTopInch:    0.4,
			MarginBottomInch: 0.4,
			MarginLeftInch:   0.4,
			MarginRightInch:  0.4,
			Landscape:        false,
		},
	}
}

// ConvertHTMLToPDF 将HTML内容转换为PDF
func (c *ChromeDPConverter) ConvertHTMLToPDF(ctx context.Context, html string, opts ...Option) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// 应用选项
	options := c.DefaultOptions
	for _, opt := range opts {
		opt(&options)
	}

	// 设置超时
	timeoutCtx, cancel := context.WithTimeout(ctx, c.DefaultTimeout)
	defer cancel()

	// 创建远程分配器
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(timeoutCtx, c.RemoteWebSocketURL)
	defer allocCancel()

	// 创建新的浏览器上下文
	taskCtx, taskCancel := chromedp.NewContext(allocCtx)
	defer taskCancel()

	// 准备PDF参数
	printToPDFParams := page.PrintToPDF().
		WithPrintBackground(true).
		WithPreferCSSPageSize(true).
		WithMarginTop(options.MarginTopInch).
		WithMarginBottom(options.MarginBottomInch).
		WithMarginLeft(options.MarginLeftInch).
		WithMarginRight(options.MarginRightInch).
		WithLandscape(options.Landscape)

	// 如果设置了纸张尺寸
	if options.PaperWidthInch > 0 && options.PaperHeightInch > 0 {
		printToPDFParams = printToPDFParams.
			WithPaperWidth(options.PaperWidthInch).
			WithPaperHeight(options.PaperHeightInch)
	}

	// 设置标题（如果提供）
	if options.Title != "" {
		html = fmt.Sprintf("<html><head><title>%s</title></head><body>%s</body></html>", options.Title, html)
	}

	var pdfData []byte
	err := chromedp.Run(taskCtx,
		// 设置HTML内容
		chromedp.Navigate("about:blank"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			frameTree, err := page.GetFrameTree().Do(ctx)
			if err != nil {
				return err
			}
			return page.SetDocumentContent(frameTree.Frame.ID, html).Do(ctx)
		}),
		// 等待页面加载完成
		chromedp.WaitReady("body", chromedp.ByQuery),
		// 生成PDF
		chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			pdfData, _, err = printToPDFParams.Do(ctx)
			return err
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("chromedp PDF生成失败: %w", err)
	}

	return pdfData, nil
}
