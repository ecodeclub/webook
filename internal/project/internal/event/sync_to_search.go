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

package event

import (
	"encoding/json"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/project/internal/domain"
)

const (
	SyncTopic = "sync_data_to_search"
)

type SyncProjectToSearchEvent struct {
	Biz   string `json:"biz"`
	BizID int64  `json:"bizID"`
	// Data 是 project 利用 json 格式序列化出来的。
	Data string `json:"data"`
}

func NewSyncProjectToSearchEvent(p domain.Project) SyncProjectToSearchEvent {
	prj := Project{
		Id:     p.Id,
		Title:  p.Title,
		Status: p.Status.ToUint8(),
		Desc:   p.Desc,
		Labels: p.Labels,
		Utime:  p.Utime,
		Resumes: slice.Map(p.Resumes, func(idx int, src domain.Resume) Resume {
			return newResume(src)
		}),
		Difficulties: slice.Map(p.Difficulties, func(idx int, src domain.Difficulty) Difficulty {
			return newDifficulty(src)
		}),
		Questions: slice.Map(p.Questions, func(idx int, src domain.Question) Question {
			return newQuestion(src)
		}),
		Introductions: slice.Map(p.Introductions, func(idx int, src domain.Introduction) Introduction {
			return newIntroduction(src)
		}),
	}
	val, _ := json.Marshal(prj)
	return SyncProjectToSearchEvent{
		Biz:   "project",
		BizID: p.Id,
		Data:  string(val),
	}
}

func newDifficulty(d domain.Difficulty) Difficulty {
	return Difficulty{
		Id:       d.Id,
		Title:    d.Title,
		Analysis: d.Analysis,
		Status:   d.Status.ToUint8(),
		Content:  d.Content,
		Utime:    d.Utime.UnixMilli(),
	}
}

func newIntroduction(p domain.Introduction) Introduction {
	return Introduction{
		Id:       p.Id,
		Role:     p.Role,
		Content:  p.Content,
		Analysis: p.Analysis,
		Status:   p.Status.ToUint8(),
		Utime:    p.Utime.UnixMilli(),
	}
}

func newResume(p domain.Resume) Resume {
	return Resume{
		Id:       p.Id,
		Role:     p.Role,
		Content:  p.Content,
		Analysis: p.Analysis,
		Status:   p.Status.ToUint8(),
		Utime:    p.Utime.UnixMilli(),
	}
}

func newQuestion(q domain.Question) Question {
	return Question{
		Id:       q.Id,
		Title:    q.Title,
		Answer:   q.Answer,
		Analysis: q.Analysis,
		Status:   q.Status.ToUint8(),
		Utime:    q.Utime.UnixMilli(),
	}
}

type Project struct {
	Id            int64          `json:"id,omitempty"`
	Title         string         `json:"title,omitempty"`
	Status        uint8          `json:"status,omitempty"`
	Desc          string         `json:"desc,omitempty"`
	Labels        []string       `json:"labels,omitempty"`
	Utime         int64          `json:"utime,omitempty"`
	Difficulties  []Difficulty   `json:"difficulties,omitempty"`
	Resumes       []Resume       `json:"resumes,omitempty"`
	Questions     []Question     `json:"questions,omitempty"`
	Introductions []Introduction `json:"introductions,omitempty"`
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
	// 这是面试时候的介绍这个项目难点
	Content string `json:"content,omitempty"`
	Status  uint8  `json:"status,omitempty"`
	Utime   int64  `json:"utime,omitempty"`
}

type Question struct {
	Id       int64  `json:"id,omitempty"`
	Title    string `json:"title,omitempty"`
	Analysis string `json:"analysis,omitempty"`
	Answer   string `json:"answer,omitempty"`
	Utime    int64  `json:"utime,omitempty"`
	Status   uint8  `json:"status"`
}

type Introduction struct {
	Id       int64  `json:"id,omitempty"`
	Role     uint8  `json:"role,omitempty"`
	Content  string `json:"content,omitempty"`
	Analysis string `json:"analysis,omitempty"`
	Status   uint8  `json:"status,omitempty"`
	Utime    int64  `json:"utime,omitempty"`
}
