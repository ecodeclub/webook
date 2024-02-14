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
	Id int64
	// 面试标题
	Title string
	// 面试题目内容
	Content string
	Utime   string

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
	Id int64

	Content string
	// 关键字，辅助记忆，提取重点
	Keywords string
	// 速记，口诀
	Shorthand string

	// 亮点
	Highlight string

	// 引导点
	Guidance string
}
