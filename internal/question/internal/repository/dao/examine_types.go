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

// ExamineRecord 业务层面上记录
type ExamineRecord struct {
	Id  int64
	Uid int64
	Qid int64
	// 代表这一次测试的 ID
	// 这个主要是为了和 AI 打交道，有一个唯一凭证
	Tid    string
	Result uint8
	// 原始的 AI 回答
	RawResult string
	// 冗余字段，使用的 tokens 数量
	Tokens int64
	// 冗余字段，花费的金额
	Amount int64

	Ctime int64
	Utime int64
}

// QuestionResult 某人是否已经回答出来了
type QuestionResult struct {
	Id int64
	// 目前来看，查询至少会有一个 uid，所以我们把 uid 放在唯一索引最前面
	Uid    int64 `gorm:"uniqueIndex:uid_qid"`
	Qid    int64 `gorm:"uniqueIndex:uid_qid"`
	Result uint8
	Ctime  int64
	Utime  int64
}
