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

type Project struct {
	// 其它字段
	Difficulties []Difficulty
	Questions    []Question
	Resumes      []Resume
}

type Difficulty struct {
}

type Resume struct {
}

// Question 虽然都叫做 Question
// 但是实际上，这个 Question 和 question 模块里面的是不同的
type Question struct {
}

type ProjectStatus uint8

func (s ProjectStatus) ToUint8() uint8 {
	return uint8(s)
}

type Role uint8

const (
	RoleUnknown = iota
	RoleStudent
	RoleIntern
	RoleCoreDeveloper
	RoleManager
)
