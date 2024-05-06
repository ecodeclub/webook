// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package web

import (
	"time"

	"github.com/ecodeclub/webook/internal/search/internal/domain"
)

type SearchReq struct {
	KeyWords string `json:"keyWords,omitempty"`
}

type Case struct {
	Id        int64    `json:"id,omitempty"`
	Uid       int64    `json:"uid,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	Title     string   `json:"title,omitempty"`
	Content   string   `json:"content,omitempty"`
	CodeRepo  string   `json:"code_repo,omitempty"`
	Keywords  string   `json:"keywords,omitempty"`
	Shorthand string   `json:"shorthand,omitempty"`
	Highlight string   `json:"highlight,omitempty"`
	Guidance  string   `json:"guidance,omitempty"`
	Status    uint8    `json:"status,omitempty"`
	Ctime     string   `json:"ctime,omitempty"`
	Utime     string   `json:"utime,omitempty"`
}

type Question struct {
	ID      int64    `json:"id,omitempty"`
	UID     int64    `json:"uid,omitempty"`
	Title   string   `json:"title,omitempty"`
	Labels  []string `json:"labels,omitempty"`
	Content string   `json:"content,omitempty"`
	Status  uint8    `json:"status,omitempty"`
	Answer  Answer   `json:"answer,omitempty"`
	Utime   string   `json:"utime,omitempty"`
}

type Answer struct {
	Analysis     AnswerElement `json:"analysis,omitempty"`
	Basic        AnswerElement `json:"basic,omitempty"`
	Intermediate AnswerElement `json:"intermediate,omitempty"`
	Advanced     AnswerElement `json:"advanced,omitempty"`
}

type AnswerElement struct {
	ID        int64  `json:"id,omitempty"`
	Content   string `json:"content,omitempty"`
	Keywords  string `json:"keywords,omitempty"`
	Shorthand string `json:"shorthand,omitempty"`
	Highlight string `json:"highlight,omitempty"`
	Guidance  string `json:"guidance,omitempty"`
}

type SkillLevel struct {
	ID        int64   `json:"id,omitempty"`
	Desc      string  `json:"desc,omitempty"`
	Ctime     string  `json:"ctime,omitempty"`
	Utime     string  `json:"utime,omitempty"`
	Questions []int64 `json:"questions,omitempty"`
	Cases     []int64 `json:"cases,omitempty"`
}

type Skill struct {
	ID           int64      `json:"id,omitempty"`
	Labels       []string   `json:"labels,omitempty"`
	Name         string     `json:"name,omitempty"`
	Desc         string     `json:"desc,omitempty"`
	Basic        SkillLevel `json:"basic,omitempty"`
	Intermediate SkillLevel `json:"intermediate,omitempty"`
	Advanced     SkillLevel `json:"advanced,omitempty"`
	Ctime        string     `json:"ctime,omitempty"`
	Utime        string     `json:"utime,omitempty"`
}

type QuestionSet struct {
	Id          int64   `json:"id,omitempty"`
	Uid         int64   `json:"uid,omitempty"`
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Questions   []int64 `json:"questions,omitempty"`
	Utime       string  `json:"utime,omitempty"`
}

type SearchResult struct {
	Cases       []Case        `json:"cases,omitempty"`
	Questions   []Question    `json:"questions,omitempty"`
	Skills      []Skill       `json:"skills,omitempty"`
	QuestionSet []QuestionSet `json:"question_set,omitempty"`
}

func NewSearchResult(res *domain.SearchResult) SearchResult {
	var newResult SearchResult
	for _, oldCase := range res.Cases {
		newCase := Case{
			Id:        oldCase.Id,
			Uid:       oldCase.Uid,
			Labels:    oldCase.Labels,
			Title:     oldCase.Title,
			Content:   oldCase.Content,
			CodeRepo:  oldCase.CodeRepo,
			Keywords:  oldCase.Keywords,
			Shorthand: oldCase.Shorthand,
			Highlight: oldCase.Highlight,
			Guidance:  oldCase.Guidance,
			Status:    oldCase.Status.ToUint8(),
			Ctime:     oldCase.Ctime.Format(time.DateTime),
			Utime:     oldCase.Utime.Format(time.DateTime),
		}
		newResult.Cases = append(newResult.Cases, newCase)
	}
	for _, question := range res.Questions {
		newQuestion := Question{
			ID:      question.ID,
			UID:     question.UID,
			Title:   question.Title,
			Labels:  question.Labels,
			Content: question.Content,
			Status:  question.Status,
			Answer: Answer{
				Analysis:     NewAnsElement(question.Answer.Analysis),
				Basic:        NewAnsElement(question.Answer.Basic),
				Intermediate: NewAnsElement(question.Answer.Intermediate),
				Advanced:     NewAnsElement(question.Answer.Advanced),
			},
			Utime: question.Utime.Format(time.DateTime),
		}
		newResult.Questions = append(newResult.Questions, newQuestion)
	}

	for _, skill := range res.Skills {
		newSkill := Skill{
			ID:           skill.ID,
			Labels:       skill.Labels,
			Name:         skill.Name,
			Desc:         skill.Desc,
			Basic:        NewSkillLevel(skill.Basic),
			Intermediate: NewSkillLevel(skill.Intermediate),
			Advanced:     NewSkillLevel(skill.Advanced),
			Ctime:        skill.Ctime.Format(time.DateTime),
			Utime:        skill.Utime.Format(time.DateTime),
		}
		newResult.Skills = append(newResult.Skills, newSkill)
	}
	for _, oldQuestionSet := range res.QuestionSet {
		newQuestionSet := QuestionSet{
			Id:          oldQuestionSet.Id,
			Uid:         oldQuestionSet.Uid,
			Title:       oldQuestionSet.Title,
			Description: oldQuestionSet.Description,
			Questions:   oldQuestionSet.Questions,
			Utime:       oldQuestionSet.Utime.Format(time.DateTime),
		}
		newResult.QuestionSet = append(newResult.QuestionSet, newQuestionSet)
	}

	return newResult
}

func NewAnsElement(ele domain.AnswerElement) AnswerElement {
	return AnswerElement{
		ID:        ele.ID,
		Content:   ele.Content,
		Keywords:  ele.Keywords,
		Shorthand: ele.Shorthand,
		Highlight: ele.Highlight,
		Guidance:  ele.Guidance,
	}
}
func NewSkillLevel(l domain.SkillLevel) SkillLevel {
	return SkillLevel{
		ID:        l.ID,
		Desc:      l.Desc,
		Ctime:     l.Ctime.Format(time.DateTime),
		Utime:     l.Utime.Format(time.DateTime),
		Questions: l.Questions,
		Cases:     l.Cases,
	}
}
