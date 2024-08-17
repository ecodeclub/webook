package web

import (
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/cases"
	baguwen "github.com/ecodeclub/webook/internal/question"

	"github.com/ecodeclub/webook/internal/skill/internal/domain"
)

type SaveReq struct {
	Skill Skill `json:"skill,omitempty"`
}

type Skill struct {
	ID           int64      `json:"id,omitempty"`
	Labels       []string   `json:"labels,omitempty"`
	Name         string     `json:"name,omitempty"`
	Desc         string     `json:"desc,omitempty"`
	Basic        SkillLevel `json:"basic,omitempty"`
	Intermediate SkillLevel `json:"intermediate,omitempty"`
	Advanced     SkillLevel `json:"advanced,omitempty"`
	Utime        string     `json:"utime,omitempty"`
}

type QuestionSet struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
}

type CaseSet struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Cases []Case `json:"cases"`
}

type SkillLevel struct {
	Id           int64         `json:"id,omitempty"`
	Desc         string        `json:"desc,omitempty"`
	Questions    []Question    `json:"questions"`
	Cases        []Case        `json:"cases"`
	QuestionSets []QuestionSet `json:"questionSets"`
	CaseSets     []CaseSet     `json:"caseSets"`
}

func (s SkillLevel) toDomain() domain.SkillLevel {
	return domain.SkillLevel{
		Id:   s.Id,
		Desc: s.Desc,
		Questions: slice.Map(s.Questions, func(idx int, src Question) int64 {
			return src.Id
		}),
		Cases: slice.Map(s.Cases, func(idx int, src Case) int64 {
			return src.Id
		}),
		QuestionSets: slice.Map(s.QuestionSets, func(idx int, src QuestionSet) int64 {
			return src.ID
		}),
		CaseSets: slice.Map(s.CaseSets, func(idx int, src CaseSet) int64 {
			return src.ID
		}),
	}
}

func (s *SkillLevel) setCases(qm map[int64]cases.Case) {
	s.Cases = slice.Map(s.Cases, func(idx int, src Case) Case {
		src.Title = qm[src.Id].Title
		return src
	})
}

func (s *SkillLevel) setQuestions(qm map[int64]baguwen.Question) {
	s.Questions = slice.Map(s.Questions, func(idx int, src Question) Question {
		src.Title = qm[src.Id].Title
		return src
	})
}

func (s *SkillLevel) setQuestionsWithExam(qm map[int64]baguwen.Question, resultMap map[int64]baguwen.ExamResult) {
	s.Questions = slice.Map(s.Questions, func(idx int, src Question) Question {
		src.Title = qm[src.Id].Title
		src.ExamineResult = resultMap[src.Id].Result.ToUint8()
		return src
	})
}

func (s *SkillLevel) setQuestionSet(qsm map[int64]baguwen.QuestionSet, resultMap map[int64]baguwen.ExamResult) {
	s.QuestionSets = slice.Map(s.QuestionSets, func(idx int, src QuestionSet) QuestionSet {
		qs := qsm[src.ID]
		src.Title = qs.Title
		res := make([]Question, 0, len(src.Questions))
		for _, q := range qs.Questions {
			exam := resultMap[q.Id]
			res = append(res, Question{
				Id:            q.Id,
				Title:         q.Title,
				ExamineResult: exam.Result.ToUint8(),
			})
		}
		src.Questions = res
		return src
	})
}

func (s *SkillLevel) setCaseSet(csm map[int64]cases.CaseSet) {
	s.CaseSets = slice.Map(s.CaseSets, func(idx int, src CaseSet) CaseSet {
		cs := csm[src.ID]
		src.Title = cs.Title
		res := make([]Case, 0, len(src.Cases))
		for _, q := range cs.Cases {
			res = append(res, Case{
				Id:    q.Id,
				Title: q.Title,
			})
		}
		src.Cases = res
		return src
	})
}

type LevelInfoReq struct {
	ID int64 `json:"id"`
}

type Sid struct {
	Sid int64 `json:"sid"`
}
type Page struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

type SkillList struct {
	Skills []Skill `json:"skills,omitempty"`
	Total  int64   `json:"total,omitempty"`
}

func (s Skill) toDomain() domain.Skill {
	skill := domain.Skill{
		ID:     s.ID,
		Labels: s.Labels,
		Name:   s.Name,
		Desc:   s.Desc,
	}
	skill.Basic = s.Basic.toDomain()
	skill.Intermediate = s.Intermediate.toDomain()
	skill.Advanced = s.Advanced.toDomain()
	return skill
}

func newSkill(s domain.Skill) Skill {
	res := Skill{
		ID:           s.ID,
		Labels:       s.Labels,
		Name:         s.Name,
		Desc:         s.Desc,
		Basic:        newSkillLevel(s.Basic),
		Intermediate: newSkillLevel(s.Intermediate),
		Advanced:     newSkillLevel(s.Advanced),
		Utime:        s.Utime.Format(time.DateTime),
	}
	return res
}

func (s *Skill) setQuestionSets(qm map[int64]baguwen.QuestionSet) {
	res := map[int64]baguwen.ExamResult{}
	s.Basic.setQuestionSet(qm, res)
	s.Intermediate.setQuestionSet(qm, res)
	s.Advanced.setQuestionSet(qm, res)
}

func (s *Skill) setCaseSets(qm map[int64]cases.CaseSet) {
	s.Basic.setCaseSet(qm)
	s.Intermediate.setCaseSet(qm)
	s.Advanced.setCaseSet(qm)
}

func (s *Skill) setQuestions(qm map[int64]baguwen.Question) {
	s.Basic.setQuestions(qm)
	s.Intermediate.setQuestions(qm)
	s.Advanced.setQuestions(qm)
}

func (s *Skill) setCases(qm map[int64]cases.Case) {
	s.Basic.setCases(qm)
	s.Intermediate.setCases(qm)
	s.Advanced.setCases(qm)
}

func newSkillLevel(s domain.SkillLevel) SkillLevel {
	return SkillLevel{
		Id:   s.Id,
		Desc: s.Desc,
		Questions: slice.Map(s.Questions, func(idx int, src int64) Question {
			return Question{
				Id: src,
			}
		}),
		Cases: slice.Map(s.Cases, func(idx int, src int64) Case {
			return Case{
				Id: src,
			}
		}),
		QuestionSets: slice.Map(s.QuestionSets, func(idx int, src int64) QuestionSet {
			return QuestionSet{
				ID: src,
			}
		}),
		CaseSets: slice.Map(s.CaseSets, func(idx int, src int64) CaseSet {
			return CaseSet{
				ID: src,
			}
		}),
	}
}

type Question struct {
	Id            int64  `json:"id,omitempty"`
	Title         string `json:"title,omitempty"`
	ExamineResult uint8  `json:"examineResult,omitempty"`
}

type Case struct {
	Id    int64  `json:"id,omitempty"`
	Title string `json:"title,omitempty"`
}

type IDs struct {
	IDs []int64 `json:"ids,omitempty"`
}
