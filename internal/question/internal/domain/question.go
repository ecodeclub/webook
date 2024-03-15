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

package domain

import "time"

// Question 和 QuestionSet 是一个多对多的关系
type Question struct {
	Id    int64
	Uid   int64
	Title string
	// 属于系统标签
	Labels  []string
	Content string

	Answer Answer
	Utime  time.Time
}

type Answer struct {
	Analysis AnswerElement
	// 基本回答
	Basic        AnswerElement
	Intermediate AnswerElement
	Advanced     AnswerElement

	Utime time.Time
}

type AnswerElement struct {
	Id      int64
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
