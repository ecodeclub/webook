package web

import "github.com/ecodeclub/webook/internal/cases/internal/domain"

type Page struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}
type CasesList struct {
	Cases []Case `json:"questions,omitempty"`
	Total int64  `json:"total,omitempty"`
}
type Case struct {
	Id  int64 `json:"id,omitempty"`
	UID int64 `json:"uid,omitempty"`
	// 面试标题
	Title  string   `json:"title,omitempty"`
	Labels []string `json:"labels,omitempty"`
	// 面试题目内容
	Content  string `json:"content,omitempty"`
	CodeRepo string `json:"codeRepo,omitempty"`
	Utime    string `json:"utime,omitempty"`

	// 面试总结
	Summary Summary `json:"summary,omitempty"`
}

type Summary struct {
	// 关键字，辅助记忆，提取重点
	Keywords string `json:"keywords,omitempty"`
	// 速记，口诀
	Shorthand string `json:"shorthand,omitempty"`
	// 亮点
	Highlight string `json:"highlight,omitempty"`
	// 引导点
	Guidance string `json:"guidance,omitempty"`
}
type CaseId struct {
	CaseId int64 `json:"caseId"`
}
type SaveReq struct {
	Case Case `json:"case,omitempty"`
}

func (c Case) toDomain() domain.Case {
	return domain.Case{
		Id:       c.Id,
		Title:    c.Title,
		Labels:   c.Labels,
		Content:  c.Content,
		CodeRepo: c.CodeRepo,
		Summary: domain.Summary{
			Keywords:  c.Summary.Keywords,
			Shorthand: c.Summary.Shorthand,
			Highlight: c.Summary.Highlight,
			Guidance:  c.Summary.Guidance,
		},
	}
}
