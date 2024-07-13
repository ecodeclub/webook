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

import "github.com/ecodeclub/ekit/sqlx"

type Question struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 作者
	Uid int64 `gorm:"index"`

	Labels sqlx.JsonColumn[[]string] `gorm:"type:varchar(512)"`
	// 面试标题
	Title string `gorm:"type=varchar(512)"`
	// 面试题目内容
	Content string

	// biz。也就是我们这个面试题目还可以用在特定一些场景下
	// 不必设置唯一索引，因为我们允许一个业务有多个题集
	// 如果 biz 和 bizId 都是零值，那么代表的就是八股文
	Biz   string `gorm:"type=varchar(256);index:biz;not null;default:'baguwen';"`
	BizId int64  `gorm:"index:biz;not null;default:0;"`

	Status uint8 `gorm:"type:tinyint(3);comment:0-未知 1-未发表 2-已发表"`
	Ctime  int64
	Utime  int64 `gorm:"index"`
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

// QuestionSet 题集
type QuestionSet struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 所有者
	Uid int64 `gorm:"index"`
	// 题集标题
	Title string
	// 题集描述
	Description string

	// 注意，QuestionSet 的 Biz 和 BizId 可以和内部的 Question 的不同
	// 举个例子来说，一个面试项目的模拟面试题，一部分是面试项目本身的题目，一部分是八股文
	Biz   string `gorm:"type=varchar(256);index:biz;not null;default:'baguwen';"`
	BizId int64  `gorm:"index:biz;not null;default:0;"`

	Ctime int64
	Utime int64 `gorm:"index"`
}

// QuestionSetQuestion 题集问题 —— 题集与题目的关联关系
type QuestionSetQuestion struct {
	Id    int64 `gorm:"primaryKey,autoIncrement"`
	QSID  int64 `gorm:"column:qs_id;uniqueIndex:qsid_qid"`
	QID   int64 `gorm:"column:qid;uniqueIndex:qsid_qid"`
	Ctime int64
	Utime int64 `gorm:"index"`
}

const (
	AnswerElementTypeUnknown = iota
	AnswerElementTypeAnalysis
	AnswerElementTypeBasic
	AnswerElementTypeIntermedia
	AnswerElementTypeAdvanced
)
