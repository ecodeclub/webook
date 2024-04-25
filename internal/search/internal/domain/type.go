package domain

import "time"

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
	Status   CaseStatus
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

type Question struct {
	ID      int64
	UID     int64
	Title   string
	Labels  []string
	Content string
	Status  uint8
	Answer  Answer
	Utime   time.Time
}

type Answer struct {
	Analysis     AnswerElement
	Basic        AnswerElement
	Intermediate AnswerElement
	Advanced     AnswerElement
}
type QuestionStatus uint8

func (s QuestionStatus) ToUint8() uint8 {
	return uint8(s)
}

type AnswerElement struct {
	ID        int64
	Content   string
	Keywords  string
	Shorthand string
	Highlight string
	Guidance  string
	Utime     time.Time
}

type SkillLevel struct {
	ID        int64
	Desc      string
	Ctime     time.Time
	Utime     time.Time
	Questions []int64
	Cases     []int64
}

type Skill struct {
	ID           int64
	Labels       []string
	Name         string
	Desc         string
	Basic        SkillLevel
	Intermediate SkillLevel
	Advanced     SkillLevel
	Ctime        time.Time
	Utime        time.Time
}

type QuestionSet struct {
	Id  int64
	Uid int64
	// 标题
	Title string
	// 描述
	Description string

	// 题集中引用的题目,
	Questions []int64
	Utime     time.Time
}

type SearchResult struct {
	Cases       []Case
	Questions   []Question
	Skills      []Skill
	QuestionSet []QuestionSet
}
