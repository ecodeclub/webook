package pdf

// WithPaperSize 设置纸张尺寸（英寸）
func WithPaperSize(width, height float64) Option {
	return func(o *Options) {
		o.PaperWidthInch = width
		o.PaperHeightInch = height
	}
}

// WithMargins 设置页边距（英寸）
func WithMargins(top, right, bottom, left float64) Option {
	return func(o *Options) {
		o.MarginTopInch = top
		o.MarginRightInch = right
		o.MarginBottomInch = bottom
		o.MarginLeftInch = left
	}
}

// WithLandscape 设置横向打印
func WithLandscape(landscape bool) Option {
	return func(o *Options) {
		o.Landscape = landscape
	}
}

// WithTitle 设置PDF标题
func WithTitle(title string) Option {
	return func(o *Options) {
		o.Title = title
	}
}

// 预定义纸张尺寸（英寸）
var (
	// A4纸尺寸（8.27 x 11.69英寸）
	PaperA4 = WithPaperSize(8.27, 11.69)
	// Letter尺寸（8.5 x 11英寸）
	PaperLetter = WithPaperSize(8.5, 11)
	// Legal尺寸（8.5 x 14英寸）
	PaperLegal = WithPaperSize(8.5, 14)
	// A3纸尺寸（11.69 x 16.54英寸）
	PaperA3 = WithPaperSize(11.69, 16.54)
)

// 预定义边距
var (
	// 标准边距（0.4英寸）
	MarginsNormal = WithMargins(0.4, 0.4, 0.4, 0.4)
	// 窄边距（0.2英寸）
	MarginsNarrow = WithMargins(0.2, 0.2, 0.2, 0.2)
	// 宽边距（0.75英寸）
	MarginsWide = WithMargins(0.75, 0.75, 0.75, 0.75)
	// 无边距
	MarginsNone = WithMargins(0, 0, 0, 0)
)
