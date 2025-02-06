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
	"github.com/ecodeclub/webook/internal/interactive"
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
	Utime   int64  `json:"utime,omitempty"`
	Status  uint8  `json:"status,omitempty"`
	Biz     string `json:"biz"`
	BizId   int64  `json:"bizId"`

	Analysis AnswerElement `json:"analysis,omitempty"`
	// 基本回答
	Basic AnswerElement `json:"basic,omitempty"`
	// 进阶回答
	Intermediate AnswerElement `json:"intermediate,omitempty"`
	// 高阶回答
	Advanced    AnswerElement `json:"advanced,omitempty"`
	Interactive Interactive   `json:"interactive"`

	ExamineResult uint8 `json:"examineResult"`

	// 是否有权限
	Permitted bool `json:"permitted"`
}

func (que Question) toDomain() domain.Question {
	return domain.Question{
		Id:      que.Id,
		Title:   que.Title,
		Content: que.Content,
		Labels:  que.Labels,
		Biz:     que.Biz,
		BizId:   que.BizId,
		Answer: domain.Answer{
			Analysis:     que.Analysis.toDomain(),
			Basic:        que.Basic.toDomain(),
			Intermediate: que.Intermediate.toDomain(),
			Advanced:     que.Advanced.toDomain(),
		},
	}
}

func newQuestion(que domain.Question, intr interactive.Interactive) Question {
	return Question{
		Id:           que.Id,
		Title:        que.Title,
		Content:      que.Content,
		Labels:       que.Labels,
		Biz:          que.Biz,
		BizId:        que.BizId,
		Status:       que.Status.ToUint8(),
		Analysis:     newAnswerElement(que.Answer.Analysis),
		Basic:        newAnswerElement(que.Answer.Basic),
		Intermediate: newAnswerElement(que.Answer.Intermediate),
		Advanced:     newAnswerElement(que.Answer.Advanced),
		Utime:        que.Utime.UnixMilli(),
		Interactive:  newInteractive(intr),
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

type CandidateReq struct {
	QSID   int64 `json:"qsid"`
	Offset int   `json:"offset,omitempty"`
	Limit  int   `json:"limit,omitempty"`
}

type Qid struct {
	Qid int64 `json:"qid"`
}

type QuestionList struct {
	Questions []Question `json:"questions,omitempty"`
	Total     int64      `json:"total,omitempty"`
}

type UpdateQuestions struct {
	QSID int64   `json:"qsid"`
	QIDs []int64 `json:"qids,omitempty"`
}

type QuestionSetID struct {
	QSID int64 `json:"qsid"`
}

type QuestionSet struct {
	Id          int64       `json:"id,omitempty"`
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Questions   []Question  `json:"questions,omitempty"`
	Biz         string      `json:"biz"`
	BizId       int64       `json:"bizId"`
	Utime       int64       `json:"utime,omitempty"`
	Interactive Interactive `json:"interactive,omitempty"`
}

// newQuestionSet 只包含基础信息
func newQuestionSet(set domain.QuestionSet) QuestionSet {
	return QuestionSet{
		Id:          set.Id,
		Title:       set.Title,
		Description: set.Description,
		Biz:         set.Biz,
		BizId:       set.BizId,
		Utime:       set.Utime.UnixMilli(),
	}
}

type QuestionSetList struct {
	Total        int64         `json:"total,omitempty"`
	QuestionSets []QuestionSet `json:"questionSets,omitempty"`
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

type BizReq struct {
	Biz   string `json:"biz"`
	BizId int64  `json:"bizId"`
}
