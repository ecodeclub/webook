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

package event

import (
	"encoding/json"

	"github.com/ecodeclub/webook/internal/question/internal/domain"
)

const (
	QuestionBiz    = "question"
	QuestionSetBiz = "questionSet"
)

type QuestionEvent struct {
	Biz   string `json:"biz"`
	BizID int    `json:"bizID"`
	Data  string `json:"data"`
}
type Question struct {
	ID      int64    `json:"id"`
	UID     int64    `json:"uid"`
	Title   string   `json:"title"`
	Labels  []string `json:"labels"`
	Content string   `json:"content"`
	Status  uint8    `json:"status"`
	Answer  Answer   `json:"answer"`
	Utime   int64    `json:"utime"`
}

type Answer struct {
	Analysis     AnswerElement `json:"analysis"`
	Basic        AnswerElement `json:"basic"`
	Intermediate AnswerElement `json:"intermediate"`
	Advanced     AnswerElement `json:"advanced"`
}

type AnswerElement struct {
	ID        int64  `json:"id"`
	Content   string `json:"content"`
	Keywords  string `json:"keywords"`
	Shorthand string `json:"shorthand"`
	Highlight string `json:"highlight"`
	Guidance  string `json:"guidance"`
	Utime     int64  `json:"utime"`
}

type QuestionSet struct {
	Id          int64   `json:"id"`
	Uid         int64   `json:"uid"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Questions   []int64 `json:"questions"`
	Utime       int64   `json:"utime"`
}

func NewQuestionEvent(q *domain.Question) QuestionEvent {
	que := newQuestion(q)
	qByte, _ := json.Marshal(que)
	return QuestionEvent{
		Biz:   QuestionBiz,
		BizID: int(q.Id),
		Data:  string(qByte),
	}
}

func NewQuestionSetEvent(q domain.QuestionSet) QuestionEvent {
	que := newQuestionSet(q)
	qByte, _ := json.Marshal(que)
	return QuestionEvent{
		Biz:   QuestionSetBiz,
		BizID: int(q.Id),
		Data:  string(qByte),
	}
}
func newQuestionSet(q domain.QuestionSet) QuestionSet {
	qids := make([]int64, 0, len(q.Questions))
	for _, que := range q.Questions {
		qids = append(qids, que.Id)
	}
	return QuestionSet{
		Id:          q.Id,
		Uid:         q.Uid,
		Title:       q.Title,
		Description: q.Description,
		Utime:       q.Utime.UnixMilli(),
		Questions:   qids,
	}
}

func newQuestion(q *domain.Question) Question {
	return Question{
		ID:      q.Id,
		UID:     q.Uid,
		Title:   q.Title,
		Labels:  q.Labels,
		Content: q.Content,
		Status:  q.Status.ToUint8(),
		Answer: Answer{
			Analysis:     newAnswerElement(q.Answer.Analysis),
			Basic:        newAnswerElement(q.Answer.Basic),
			Intermediate: newAnswerElement(q.Answer.Intermediate),
			Advanced:     newAnswerElement(q.Answer.Advanced),
		},
		Utime: q.Utime.UnixMilli(),
	}

}
func newAnswerElement(ele domain.AnswerElement) AnswerElement {
	return AnswerElement{
		ID:        ele.Id,
		Content:   ele.Content,
		Keywords:  ele.Keywords,
		Shorthand: ele.Shorthand,
		Highlight: ele.Highlight,
		Guidance:  ele.Guidance,
	}
}
