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

// Experience 代表工作经历
type Experience struct {
	// 主键
	Id int64
	// 用户的 ID
	Uid int64
	// 开始时间
	Start time.Time
	// 结束时间，如果 End 是零值，代表当前还没离职
	End time.Time

	Title       string // 职位
	CompanyName string // 公司
	Location    string // 地点
	// JSON 串存起来就可以
	Responsibilities []Responsibility // 主要职责
	Accomplishments  []Accomplishment // 主要成就
	Skills           []string         // 主要技能
}

type Responsibility struct {
	// Type 是类型，比如说核心研发、团队管理
	// 用 string 来作为枚举
	Type    string
	Content string
}

type Accomplishment struct {
	// Type 是类型，比如说性能优化，获奖啥的
	Type    string
	Content string
}

// CurrentEmployed 是否当前正在职
func (e Experience) CurrentEmployed() bool {
	return e.End.IsZero()
}
