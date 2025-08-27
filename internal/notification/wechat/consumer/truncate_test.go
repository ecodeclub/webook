package consumer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 表格驱动的单元测试
func TestTruncate(t *testing.T) {
	// 定义测试用例的结构体
	testCases := []struct {
		name     string // 测试用例名称
		content  string // 输入字符串
		limit    int    // 截断长度限制（字节）
		expected string // 期望输出
	}{
		// --- 正常场景 ---
		{
			name:     "正常截断-纯ASCII",
			content:  "Hello, World!",
			limit:    5,
			expected: "Hello",
		},
		{
			name:     "正常截断-包含中文字符",
			content:  "你好，世界", // "你好"占6字节, "，"占3字节
			limit:    7,       // 7 落在'，'的第1个字节之后
			expected: "你好",    // 应该回退到'，'之前
		},
		{
			name:     "截断位置刚好在一个完整中文字符后",
			content:  "Go语言编程", // "Go语言" 占 2+3+3=8字节
			limit:    8,
			expected: "Go语言",
		},
		{
			name:     "正常截断-包含Emoji",
			content:  "Go语言很酷👍", // "👍" 占4字节
			limit:    16,        // 16 落在 "👍" 的中间
			expected: "Go语言很酷",  // 应该回退到 "👍" 之前
		},

		// --- 边界场景 ---
		{
			name:     "边界-字符串长度小于限制",
			content:  "short string",
			limit:    20,
			expected: "short string",
		},
		{
			name:     "边界-字符串长度等于限制",
			content:  "exact length",
			limit:    12,
			expected: "exact length",
		},
		{
			name:     "边界-限制为0",
			content:  "any string",
			limit:    0,
			expected: "",
		},
		{
			name:     "边界-空字符串",
			content:  "",
			limit:    10,
			expected: "",
		},
		{
			name:     "边界-截断位置刚好在一个ASCII字符后",
			content:  "boundary test",
			limit:    8,
			expected: "boundary",
		},
	}

	// 遍历所有测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 执行待测试的函数
			actual := truncate(tc.content, tc.limit)
			// 使用 testify/assert 断言结果是否符合预期
			assert.Equal(t, tc.expected, actual)
		})
	}
}

// --- 异常场景测试 ---
// 单独测试会引发 panic 的情况
func TestTruncate_Panic(t *testing.T) {
	// 断言当 limit 为负数时，程序会发生 panic
	assert.Panics(t, func() {
		_ = truncate("this will panic", -1)
	}, "limit为负数应该引起panic")
}
