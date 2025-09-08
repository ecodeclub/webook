package html_truncate

import (
	"regexp"
	"strings"
)

// StripHTML 去除HTML标签，包括超链接
func StripHTML(content string) string {
	// 先处理超链接，完全去除
	content = processLinks(content)

	// 处理HTML结构标签，在去除标签之前
	content = processHTMLStructureForStripHTML(content)

	// 去除所有剩余的HTML标签
	re := regexp.MustCompile(`<[^>]*>`)
	content = re.ReplaceAllString(content, "")

	// 处理HTML实体
	content = processHTMLEntities(content)

	// 处理多余的空白
	content = processWhitespaceForStripHTML(content)

	return content
}

// processLinks 处理超链接，完全去除超链接及其内容
func processLinks(content string) string {
	// 匹配<a>标签及其内容
	re := regexp.MustCompile(`<a\s+[^>]*>.*?</a>`)
	return re.ReplaceAllString(content, "")
}

// processHTMLEntities 处理常见的HTML实体
func processHTMLEntities(content string) string {
	replacements := map[string]string{
		"&amp;":  "&",
		"&lt;":   "<",
		"&gt;":   ">",
		"&quot;": "\"",
		"&#39;":  "'",
		"&nbsp;": " ",
	}

	for entity, replacement := range replacements {
		content = strings.ReplaceAll(content, entity, replacement)
	}

	return content
}

// processHTMLStructureForStripHTML 专门为StripHTML处理HTML结构
func processHTMLStructureForStripHTML(content string) string {
	// 处理列表项，为每个li添加项目符号和换行
	content = strings.ReplaceAll(content, "<li>", "\n• ")
	content = strings.ReplaceAll(content, "</li>", "")

	// 处理列表容器
	content = strings.ReplaceAll(content, "<ul>", "")
	content = strings.ReplaceAll(content, "</ul>", "")
	content = strings.ReplaceAll(content, "<ol>", "")
	content = strings.ReplaceAll(content, "</ol>", "")

	// 处理段落
	content = strings.ReplaceAll(content, "<p>", "")
	content = strings.ReplaceAll(content, "</p>", "\n")

	// 处理标题
	for i := 1; i <= 6; i++ {
		openTag := "<h" + string(rune('0'+i)) + ">"
		closeTag := "</h" + string(rune('0'+i)) + ">"
		content = strings.ReplaceAll(content, openTag, "")
		content = strings.ReplaceAll(content, closeTag, " ")
	}

	// 处理块引用
	content = strings.ReplaceAll(content, "<blockquote>", "")
	content = strings.ReplaceAll(content, "</blockquote>", "")

	// 处理代码块
	content = strings.ReplaceAll(content, "<pre>", "")
	content = strings.ReplaceAll(content, "</pre>", "")
	content = strings.ReplaceAll(content, "<pre data-language=\"plain\">", "")

	return content
}

// processWhitespaceForStripHTML 专门为StripHTML处理空白字符
func processWhitespaceForStripHTML(content string) string {
	// 处理代码块内的换行符 - 在代码内容中将换行符替换为空格
	// 这需要在结构处理之后进行，因为此时代码块标签已经被移除

	// 将制表符和多个空格压缩为单个空格
	re := regexp.MustCompile(`[ \t]+`)
	content = re.ReplaceAllString(content, " ")

	// 处理多个连续换行符
	re = regexp.MustCompile(`\n{3,}`)
	content = re.ReplaceAllString(content, "\n\n")

	// 对于非列表项的内容，需要特殊处理
	// 如果内容不包含"• "（列表项标记），则压缩所有换行符
	if !strings.Contains(content, "• ") {
		re = regexp.MustCompile(`\s+`)
		content = re.ReplaceAllString(content, " ")
	}

	// 修剪前后空白
	content = strings.TrimSpace(content)

	return content
}
