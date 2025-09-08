package html_truncate

import (
	"testing"
)

func TestStripHTML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "超链接",
			input:    `<p>整个案例的代码在：<a href="https://github.com/meoying/interview-cases/tree/main/case11_20/case11" rel="noopener noreferrer" target="_blank">interview-cases/case11_20/case11 at main · meoying/interview-cases (github.com)</a>，是 Go 版本。</p>`,
			expected: "整个案例的代码在：，是 Go 版本。",
		},
		{
			name:     "段落和列表",
			input:    `<p>正常的 Redis 用作缓存都是这个套路：</p><ul><li>查询的时候先查 Redis。如果 Redis 中没有数据，则查数据库，查到了就回写数据库；</li><li>更新策略则是没有各种各样，比如说先更新 DB，再删除 Redis。</li></ul>`,
			expected: "正常的 Redis 用作缓存都是这个套路：\n\n• 查询的时候先查 Redis。如果 Redis 中没有数据，则查数据库，查到了就回写数据库；\n• 更新策略则是没有各种各样，比如说先更新 DB，再删除 Redis。",
		},
		{
			name:     "嵌套列表",
			input:    `<ul><li>查询的时候先查 Redis。如果 Redis 中没有数据，则：<ul><li>如果当前请求是被限流的请求，那么直接返回，不需要再去查询数据库</li><li>如果当前请求是正常的请求，那么就回查 MySQL</li></ul></li><li>更新策略：保持不变。</li></ul>`,
			expected: "• 查询的时候先查 Redis。如果 Redis 中没有数据，则：\n• 如果当前请求是被限流的请求，那么直接返回，不需要再去查询数据库\n• 如果当前请求是正常的请求，那么就回查 MySQL\n• 更新策略：保持不变。",
		},
		{
			name:     "标题",
			input:    `<h3>代码实现</h3><p>防止你看不懂这个案例的代码，这里做一个简单的介绍。</p>`,
			expected: "代码实现 防止你看不懂这个案例的代码，这里做一个简单的介绍。",
		},
		{
			name:     "块引用",
			input:    `<blockquote>我在 XXX 业务里面结合限流设计过一个比较特别的缓存方案。</blockquote>`,
			expected: "我在 XXX 业务里面结合限流设计过一个比较特别的缓存方案。",
		},
		{
			name:     "图片",
			input:    `<p><img src="https://cdn.meoying.com/interview/c523c431-16d5-44b3-873a-dcf0bbb452cb" style="width: 50%; display: block; margin: auto;"></p>`,
			expected: "",
		},
		{
			name:     "HTML实体",
			input:    `<p>ctx = context.WithValue(ctx, &quot;RateLimited&quot;, true)</p>`,
			expected: "ctx = context.WithValue(ctx, \"RateLimited\", true)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripHTML(tt.input)
			if result != tt.expected {
				t.Errorf("StripHTML() = %v, want %v", result, tt.expected)
			}
		})
	}
}
