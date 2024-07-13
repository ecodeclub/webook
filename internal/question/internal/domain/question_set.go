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

import (
	"time"

	"github.com/ecodeclub/ekit/slice"
)

// QuestionSet 题集实体
type QuestionSet struct {
	Id  int64
	Uid int64
	// 标题
	Title string
	// 描述
	Description string

	Biz   string
	BizId int64

	// 题集中引用的题目,
	Questions []Question

	Utime time.Time
}

func (set QuestionSet) Qids() []int64 {
	return slice.Map(set.Questions, func(idx int, src Question) int64 {
		return src.Id
	})
}
