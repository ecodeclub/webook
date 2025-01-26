package html_truncate

import (
	"regexp"
	"strings"
)

type HTMLTruncator interface {
	Truncate(content string) string
}

func DefaultHTMLTruncator() HTMLTruncator {
	return htmlTruncator{}
}

type htmlTruncator struct{}

func (t htmlTruncator) Truncate(content string) string {
	parahCount := t.ParagraphCount(content)
	switch {
	case parahCount <= 3:
		return t.TruncateByParagraphs(content, 1)
	case parahCount == 4:
		return t.TruncateByParagraphs(content, 2)
	default:
		return t.TruncateByParagraphs(content, 3)
	}
}

// ParagraphCount 查看有几段
func (t htmlTruncator) ParagraphCount(content string) int {
	// 使用正则表达式匹配<p>标签及其内容
	re := regexp.MustCompile(`(<p>.*?</p>)`)
	matches := re.FindAllStringSubmatchIndex(content, -1)

	pCount := 0

	// 遍历找到的p标签位置
	for _, match := range matches {
		// 获取当前p标签的内容
		pContent := content[match[0]:match[1]]

		// 提取p标签中的文本内容
		textRe := regexp.MustCompile(`<p>(.*?)</p>`)
		textMatch := textRe.FindStringSubmatch(pContent)

		// 如果p标签不为空，则计数加1
		if len(textMatch) > 1 && strings.TrimSpace(textMatch[1]) != "" {
			pCount++
		}
	}

	return pCount
}

// TruncateByParagraphs 按段落截取HTML内容
// content: 原始HTML内容
// count: 需要保留的段落数量
func (htmlTruncator) TruncateByParagraphs(content string, number int) string {
	if number <= 0 {
		return ""
	}

	// 使用正则表达式匹配<p>标签及其内容
	re := regexp.MustCompile(`(<p>.*?</p>)`)
	matches := re.FindAllStringSubmatchIndex(content, -1)

	if len(matches) == 0 {
		return content
	}

	pCount := 0
	var lastIndex int

	// 遍历找到的p标签位置
	for _, match := range matches {
		// 获取当前p标签的内容
		pContent := content[match[0]:match[1]]

		// 提取p标签中的文本内容
		textRe := regexp.MustCompile(`<p>(.*?)</p>`)
		textMatch := textRe.FindStringSubmatch(pContent)

		// 如果p标签不为空，则计数加1
		if len(textMatch) > 1 && strings.TrimSpace(textMatch[1]) != "" {
			pCount++
			lastIndex = match[1]

			// 如果已达到指定数量，截断内容
			if pCount == number {
				return content[:lastIndex]
			}
		}
	}

	// 如果有效p标签数量少于指定数量，返回全部内容
	return content
}
