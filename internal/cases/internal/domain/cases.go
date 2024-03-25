package domain

import (
	"time"
)

type Case struct {
	Id int64
	// 作者
	Uid      int64
	Labels   []string
	Title    string
	Content  string
	CodeRepo string
	// 关键字，辅助记忆，提取重点
	Keywords string
	// 速记，口诀
	Shorthand string
	// 亮点
	Highlight string
	// 引导点
	Guidance string

	Ctime time.Time
	Utime time.Time
}
