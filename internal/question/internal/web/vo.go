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

type SaveReq struct {
	// 问题的 ID
	Question Question `json:"question"`
}

type Question struct {
	Id int64 `json:"id,omitempty"`
	// 面试标题
	Title string `json:"title,omitempty"`
	// 面试题目内容
	Content string `json:"content,omitempty"`
	Utime   string `json:"utime,omitempty"`

	Answer Answer `json:"answer,omitempty"`
}

type Answer struct {
	Analysis AnswerElement `json:"analysis,omitempty"`
	// 基本回答
	Basic AnswerElement `json:"basic,omitempty"`
	// 进阶回答
	Intermediate AnswerElement `json:"intermediate,omitempty"`
	// 高阶回答
	Advanced AnswerElement `json:"advanced,omitempty"`
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

type CreateQuestionSetReq struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

type AddQuestionsToQuestionSetReq struct {
	QSID      int64      `json:"qsid"`
	Questions []Question `json:"questions,omitempty"`
}

type DeleteQuestionsFromQuestionSetReq struct {
	QSID      int64      `json:"qsid"`
	Questions []Question `json:"questions,omitempty"`
}

type QuestionSetID struct {
	QuestionSetID int64 `json:"qsid"`
}
