package pdf

import (
	"context"
)

// Converter PDF转换器接口
//
//go:generate mockgen -source=./pdf.go -package=pdfmocks -destination=./mocks/pdf.mock.go -typed Converter
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
