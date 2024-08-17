package domain

import "time"

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

func (s Skill) Cases() []int64 {
	res := make([]int64, 0, s.caseLen())
	res = append(res, s.Basic.Cases...)
	res = append(res, s.Intermediate.Cases...)
	res = append(res, s.Advanced.Cases...)
	return res
}

func (s Skill) QuestionSets() []int64 {
	res := make([]int64, 0, s.questionSetLen())
	res = append(res, s.Basic.QuestionSets...)
	res = append(res, s.Intermediate.Questions...)
	res = append(res, s.Advanced.QuestionSets...)
	return res
}

func (s Skill) questionSetLen() int {
	return len(s.Basic.QuestionSets) + len(s.Intermediate.QuestionSets) + len(s.Advanced.QuestionSets)
}

func (s Skill) CaseSets() []int64 {
	res := make([]int64, 0, s.caseSetLen())
	res = append(res, s.Basic.CaseSets...)
	res = append(res, s.Intermediate.CaseSets...)
	res = append(res, s.Advanced.CaseSets...)
	return res
}

func (s Skill) caseSetLen() int {
	return len(s.Basic.CaseSets) + len(s.Intermediate.CaseSets) + len(s.Advanced.CaseSets)
}

func (s Skill) caseLen() int {
	return len(s.Basic.Cases) + len(s.Intermediate.Cases) + len(s.Advanced.Cases)
}

func (s Skill) Questions() []int64 {
	res := make([]int64, 0, s.questionLen())
	res = append(res, s.Basic.Questions...)
	res = append(res, s.Intermediate.Questions...)
	res = append(res, s.Advanced.Questions...)
	return res
}

func (s Skill) questionLen() int {
	return len(s.Basic.Questions) + len(s.Intermediate.Questions) + len(s.Advanced.Questions)
}

type SkillLevel struct {
	Id           int64
	Desc         string
	Ctime        time.Time
	Utime        time.Time
	Questions    []int64
	Cases        []int64
	QuestionSets []int64
	CaseSets     []int64
}
