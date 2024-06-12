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

import (
	"time"

	"github.com/ecodeclub/webook/internal/interactive"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/webook/internal/project/internal/domain"
)

type Page struct {
	Offset int `json:"offset,omitempty"`
	Limit  int `json:"limit,omitempty"`
}

type Project struct {
	Id             int64  `json:"id,omitempty"`
	SN             string `json:"sn,omitempty"`
	Title          string `json:"title,omitempty"`
	Status         uint8  `json:"status,omitempty"`
	Desc           string `json:"desc,omitempty"`
	GithubRepo     string `json:"githubRepo,omitempty"`
	GiteeRepo      string `json:"giteeRepo,omitempty"`
	RefQuestionSet int64  `json:"refQuestionSet,omitempty"`
	// 整体概况
	Overview      string         `json:"overview,omitempty"`
	SystemDesign  string         `json:"systemDesign,omitempty"`
	Labels        []string       `json:"labels,omitempty"`
	Utime         int64          `json:"utime,omitempty"`
	Difficulties  []Difficulty   `json:"difficulties,omitempty"`
	Resumes       []Resume       `json:"resumes,omitempty"`
	Questions     []Question     `json:"questions,omitempty"`
	Introductions []Introduction `json:"introductions,omitempty"`
	Combos        []Combo        `json:"combos,omitempty"`
	Interactive   Interactive    `json:"interactive,omitempty"`
	Permitted     bool           `json:"permitted"`
	CodeSPU       string         `json:"codeSPU"`
	ProductSPU    string         `json:"productSPU"`
}

func newProject(p domain.Project, intr interactive.Interactive) Project {
	return Project{
		Id:             p.Id,
		Title:          p.Title,
		SN:             p.SN,
		Overview:       p.Overview,
		SystemDesign:   p.SystemDesign,
		Status:         p.Status.ToUint8(),
		GithubRepo:     p.GithubRepo,
		GiteeRepo:      p.GiteeRepo,
		RefQuestionSet: p.RefQuestionSet,
		Desc:           p.Desc,
		Labels:         p.Labels,
		Utime:          p.Utime,
		CodeSPU:        p.CodeSPU,
		ProductSPU:     p.ProductSPU,
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
		Combos: slice.Map(p.Combos, func(idx int, src domain.Combo) Combo {
			return newCombo(src)
		}),
		Interactive: newInteractive(intr),
	}
}

func (p Project) toDomain() domain.Project {
	return domain.Project{
		Id:             p.Id,
		Title:          p.Title,
		SN:             p.SN,
		GiteeRepo:      p.GiteeRepo,
		GithubRepo:     p.GithubRepo,
		RefQuestionSet: p.RefQuestionSet,
		Status:         domain.ProjectStatus(p.Status),
		Desc:           p.Desc,
		Labels:         p.Labels,
		Overview:       p.Overview,
		SystemDesign:   p.SystemDesign,
	}
}

type Resume struct {
	Id       int64  `json:"id,omitempty"`
	Role     uint8  `json:"role,omitempty"`
	Content  string `json:"content,omitempty"`
	Analysis string `json:"analysis,omitempty"`
	Status   uint8  `json:"status,omitempty"`
	Utime    int64  `json:"utime,omitempty"`
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

func (r Resume) toDomain() domain.Resume {
	return domain.Resume{
		Id:       r.Id,
		Role:     r.Role,
		Content:  r.Content,
		Analysis: r.Analysis,
		Status:   domain.ResumeStatus(r.Status),
	}
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

func (d Difficulty) toDomain() domain.Difficulty {
	return domain.Difficulty{
		Id:       d.Id,
		Title:    d.Title,
		Analysis: d.Analysis,
		Status:   domain.DifficultyStatus(d.Status),
		Content:  d.Content,
	}
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

type ResumeList struct {
	Resumes []Resume
	Total   int64
}

type PidPage struct {
	Pid int64 `json:"pid"`
	Page
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
	Status   uint8  `json:"status"`
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

func (q Question) toDomain() domain.Question {
	return domain.Question{
		Id:       q.Id,
		Title:    q.Title,
		Answer:   q.Answer,
		Analysis: q.Analysis,
		Status:   domain.QuestionStatus(q.Status),
		Utime:    time.UnixMilli(q.Utime),
	}
}

type Introduction struct {
	Id       int64  `json:"id,omitempty"`
	Role     uint8  `json:"role,omitempty"`
	Content  string `json:"content,omitempty"`
	Analysis string `json:"analysis,omitempty"`
	Status   uint8  `json:"status,omitempty"`
	Utime    int64  `json:"utime,omitempty"`
}

type IntroductionSaveReq struct {
	Pid          int64        `json:"pid"`
	Introduction Introduction `json:"introduction"`
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

func (r Introduction) toDomain() domain.Introduction {
	return domain.Introduction{
		Id:       r.Id,
		Role:     r.Role,
		Content:  r.Content,
		Analysis: r.Analysis,
		Status:   domain.IntroductionStatus(r.Status),
	}
}

type ProjectList struct {
	Total    int64     `json:"total,omitempty"`
	Projects []Project `json:"projects,omitempty"`
}

// IdReq Admin 端一般操作 id
type IdReq struct {
	Id int64 `json:"id,omitempty"`
}

type Interactive struct {
	CollectCnt int  `json:"collectCnt"`
	LikeCnt    int  `json:"likeCnt"`
	ViewCnt    int  `json:"viewCnt"`
	Liked      bool `json:"liked"`
	Collected  bool `json:"collected"`
}

func newInteractive(intr interactive.Interactive) Interactive {
	return Interactive{
		CollectCnt: intr.CollectCnt,
		ViewCnt:    intr.ViewCnt,
		LikeCnt:    intr.LikeCnt,
		Liked:      intr.Liked,
		Collected:  intr.Collected,
	}
}

// Combo 面试套路（连招）
type Combo struct {
	Id      int64  `json:"id,omitempty"`
	Title   string `json:"title,omitempty"`
	Content string `json:"content,omitempty"`
	Utime   int64  `json:"utime,omitempty"`
	Status  uint8  `json:"status,omitempty"`
}

func (c Combo) toDomain() domain.Combo {
	return domain.Combo{
		Id:      c.Id,
		Title:   c.Title,
		Content: c.Content,
		Utime:   c.Utime,
		Status:  domain.ComboStatus(c.Status),
	}
}

func newCombo(c domain.Combo) Combo {
	return Combo{
		Id:      c.Id,
		Title:   c.Title,
		Content: c.Content,
		Utime:   c.Utime,
		Status:  c.Status.ToUint8(),
	}
}

type ComboSaveReq struct {
	Pid   int64 `json:"pid,omitempty"`
	Combo Combo `json:"combo,omitempty"`
}
