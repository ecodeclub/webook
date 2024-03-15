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

	"github.com/ecodeclub/webook/internal/question/internal/domain"
)

type SaveReq struct {
	// 问题的 ID
	Question Question `json:"question,omitempty"`
}

type Question struct {
	Id int64 `json:"id,omitempty"`
	// 面试标题
	Title  string   `json:"title,omitempty"`
	Labels []string `json:"labels,omitempty"`
	// 面试题目内容
	Content string `json:"content,omitempty"`
	Utime   string `json:"utime,omitempty"`

	// 题集 ID
	Sets []QuestionSet `json:"sets"`

	Analysis AnswerElement `json:"analysis,omitempty"`
	// 基本回答
	Basic AnswerElement `json:"basic,omitempty"`
	// 进阶回答
	Intermediate AnswerElement `json:"intermediate,omitempty"`
	// 高阶回答
	Advanced AnswerElement `json:"advanced,omitempty"`
}

func (que Question) toDomain() domain.Question {
	return domain.Question{
		Id:      que.Id,
		Title:   que.Title,
		Content: que.Content,
		Labels:  que.Labels,
		Answer: domain.Answer{
			Analysis:     que.Analysis.toDomain(),
			Basic:        que.Basic.toDomain(),
			Intermediate: que.Intermediate.toDomain(),
			Advanced:     que.Intermediate.toDomain(),
		},
	}
}

func newQuestion(que domain.Question) Question {
	return Question{
		Id:           que.Id,
		Title:        que.Title,
		Content:      que.Content,
		Labels:       que.Labels,
		Analysis:     newAnswerElement(que.Answer.Analysis),
		Basic:        newAnswerElement(que.Answer.Basic),
		Intermediate: newAnswerElement(que.Answer.Intermediate),
		Advanced:     newAnswerElement(que.Answer.Advanced),
		Utime:        que.Utime.Format(time.DateTime),
	}
}

type AnswerElement struct {
	Id int64 `json:"id,omitempty"`

	Content string `json:"content,omitempty"`
	// 关键字，辅助记忆，提取重点
	Keywords string `json:"keywords,omitempty"`
	// 速记，口诀
	Shorthand string `json:"shorthand,omitempty"`

	// 亮点
	Highlight string `json:"highlight,omitempty"`

	// 引导点
	Guidance string `json:"guidance,omitempty"`
}

func (ele AnswerElement) toDomain() domain.AnswerElement {
	return domain.AnswerElement{
		Id:        ele.Id,
		Content:   ele.Content,
		Keywords:  ele.Keywords,
		Shorthand: ele.Shorthand,
		Highlight: ele.Highlight,
		Guidance:  ele.Guidance,
	}
}

func newAnswerElement(ele domain.AnswerElement) AnswerElement {
	return AnswerElement{
		Id:        ele.Id,
		Content:   ele.Content,
		Keywords:  ele.Keywords,
		Shorthand: ele.Shorthand,
		Highlight: ele.Highlight,
		Guidance:  ele.Guidance,
	}
}

type Page struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

type Qid struct {
	Qid int64 `json:"qid"`
}

type QuestionList struct {
	Questions []Question `json:"questions,omitempty"`
	Total     int64      `json:"total,omitempty"`
}

type SaveQuestionSetReq struct {
	Id          int64  `json:"id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type UpdateQuestionsOfQuestionSetReq struct {
	QSID int64   `json:"qsid"`
	QIDs []int64 `json:"qids,omitempty"`
}

type QuestionSetID struct {
	QSID int64 `json:"qsid"`
}

type QuestionSet struct {
	Id          int64      `json:"id,omitempty"`
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	Questions   []Question `json:"questions,omitempty"`
	Utime       string     `json:"utime,omitempty"`
}

type QuestionSetList struct {
	Total        int64         `json:"total,omitempty"`
	QuestionSets []QuestionSet `json:"questionSets,omitempty"`
}
