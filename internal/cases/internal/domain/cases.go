package domain

import (
	"time"
)

const BizCase = "case"

type Case struct {
	Id int64
	// 作者
	Uid          int64
	Labels       []string
	Introduction string
	Title        string
	Content      string
	CodeRepo     string
	// 关键字，辅助记忆，提取重点
	Keywords string
	// 速记，口诀
	Shorthand string
	// 亮点
	Highlight string
	// 引导点
	Guidance string
	Status   CaseStatus
	Biz      string
	BizId    int64
	Ctime    time.Time
	Utime    time.Time
}

type CaseStatus uint8

func (s CaseStatus) ToUint8() uint8 {
	return uint8(s)
}

const (
	// UnknownStatus 未知
	UnknownStatus CaseStatus = 0
	// UnPublishedStatus 未发布
	UnPublishedStatus CaseStatus = 1
	// PublishedStatus 发布
	PublishedStatus CaseStatus = 2
)
