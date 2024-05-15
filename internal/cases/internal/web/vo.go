package web

import (
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/interactive"
)

type Page struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}
type CasesList struct {
	Cases []Case `json:"cases,omitempty"`
	Total int64  `json:"total,omitempty"`
}
type Case struct {
	Id  int64 `json:"id,omitempty"`
	UID int64 `json:"uid,omitempty"`
	// 面试案例标题
	Title string `json:"title,omitempty"`
	// 面试案例的简介
	Introduction string `json:"introduction,omitempty"`

	Labels []string `json:"labels,omitempty"`
	// 面试案例内容
	Content  string `json:"content,omitempty"`
	CodeRepo string `json:"codeRepo,omitempty"`
	// 关键字，辅助记忆，提取重点
	Keywords string `json:"keywords,omitempty"`
	// 速记，口诀
	Shorthand string `json:"shorthand,omitempty"`
	// 亮点
	Highlight string `json:"highlight,omitempty"`
	// 引导点
	Guidance string `json:"guidance,omitempty"`
	Status   uint8  `json:"status,omitempty"`
	Utime    int64  `json:"utime,omitempty"`

	Interactive Interactive `json:"interactive,omitempty"`
}

type CaseId struct {
	Cid int64 `json:"cid"`
}
type SaveReq struct {
	Case Case `json:"case,omitempty"`
}

func (c Case) toDomain() domain.Case {
	return domain.Case{
		Id:           c.Id,
		Title:        c.Title,
		Labels:       c.Labels,
		Content:      c.Content,
		CodeRepo:     c.CodeRepo,
		Keywords:     c.Keywords,
		Shorthand:    c.Shorthand,
		Introduction: c.Introduction,
		Highlight:    c.Highlight,
		Guidance:     c.Guidance,
	}
}

type Interactive struct {
	CollectCnt int  `json:"collectCnt"`
	LikeCnt    int  `json:"likeCnt"`
	ViewCnt    int  `json:"viewCnt"`
	Liked      bool `json:"liked"`
	Collected  bool `json:"collected"`
}

func newInteractive(intr interactive.Interactive) Interactive {
	return Interactive{
		CollectCnt: intr.CollectCnt,
		ViewCnt:    intr.ViewCnt,
		LikeCnt:    intr.LikeCnt,
		Liked:      intr.Liked,
		Collected:  intr.Collected,
	}
}
