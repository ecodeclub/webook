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

package dao

type Question struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 作者
	Uid int64 `gorm:"index"`
	// 面试标题
	Title string
	// 面试题目内容
	Content string

	Ctime int64
	Utime int64 `gorm:"index"`
}

type PublishQuestion Question

type PublishAnswerElement AnswerElement

// AnswerElement 回答，对于一个问题来说，回答分成好几个部分
// 这个就是代表一个部分
// 理论上来说应该要考虑引入一个叫做 Answer 的表
// 但是个人觉得目前没有必要
type AnswerElement struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 问题 ID
	Qid int64 `gorm:"uniqueIndex:qid_type"`

	Type uint8 `gorm:"uniqueIndex:qid_type"`

	// 回答内容
	Content string

	// 关键字，辅助记忆，提取重点
	Keywords string
	// 速记，口诀
	Shorthand string

	// 亮点
	Highlight string

	// 引导点
	Guidance string

	Ctime int64
	Utime int64
}

const (
	AnswerElementTypeUnknown = iota
	AnswerElementTypeAnalysis
	AnswerElementTypeBasic
	AnswerElementTypeIntermedia
	AnswerElementTypeAdvanced
)
