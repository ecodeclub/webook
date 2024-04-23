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

type Page struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

type Project struct {
	Id           int64        `json:"id,omitempty"`
	Title        string       `json:"title,omitempty"`
	Status       uint8        `json:"status,omitempty"`
	Desc         string       `json:"desc,omitempty"`
	Labels       []string     `json:"labels,omitempty"`
	Utime        int64        `json:"utime,omitempty"`
	Difficulties []Difficulty `json:"difficulties,omitempty"`
	Resumes      []Resume     `json:"resumes,omitempty"`
	Questions    []Question   `json:"questions,omitempty"`
}

type Resume struct {
	Id       int64  `json:"id,omitempty"`
	Role     uint8  `json:"role,omitempty"`
	Content  string `json:"content,omitempty"`
	Analysis string `json:"analysis,omitempty"`
	Status   uint8  `json:"status,omitempty"`
	Utime    int64  `json:"utime,omitempty"`
}

type Difficulty struct {
	Id       int64  `json:"id,omitempty"`
	Title    string `json:"title,omitempty"`
	Analysis string `json:"analysis,omitempty"`
	Status   uint8  `json:"status,omitempty"`
	Utime    int64  `json:"utime,omitempty"`
	// 这是面试时候的介绍这个项目难点
	Content string `json:"content,omitempty"`
}

type DifficultySaveReq struct {
	// 项目的 id
	Pid        int64      `json:"pid,omitempty"`
	Difficulty Difficulty `json:"difficulty,omitempty"`
}

type ResumeSaveReq struct {
	Pid    int64  `json:"pid,omitempty"`
	Resume Resume `json:"resume,omitempty"`
}

type QuestionSaveReq struct {
	Pid      int64    `json:"pid,omitempty"`
	Question Question `json:"question,omitempty"`
}

type Question struct {
	Id       int64  `json:"id,omitempty"`
	Title    string `json:"title,omitempty"`
	Analysis string `json:"analysis,omitempty"`
	Answer   string `json:"answer,omitempty"`
	Utime    int64  `json:"utime,omitempty"`
}

type ProjectList struct {
	Total    int       `json:"total,omitempty"`
	Projects []Project `json:"projects,omitempty"`
}

// IdReq Admin 端一般操作 id
type IdReq struct {
	Id int64 `json:"id,omitempty"`
}
