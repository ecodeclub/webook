package domain

import "time"

// QuestionSet 题集实体
type QuestionSet struct {
	Id  int64
	Uid int64
	// 标题
	Title string
	// 描述
	Description string

	// 题集中引用的题目,
	Questions []Question

	Utime time.Time
}
