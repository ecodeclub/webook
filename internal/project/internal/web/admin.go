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
	"fmt"
	"time"

	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
}

func NewAdminHandler() *AdminHandler {
	return &AdminHandler{}
}

func (h *AdminHandler) Routes(server *gin.Engine) {
	g := server.Group("/project")
	g.POST("/list", ginx.B[Page](h.List))
	g.POST("/detail", ginx.B[IdReq](h.Detail))
	g.POST("/save", ginx.B[Project](h.Save))
	g.POST("/publish", ginx.B[Project](h.Publish))
	g.POST("/difficulty/save", ginx.BS(h.DifficultySave))
	g.POST("/difficulty/publish", ginx.BS(h.DifficultyPublish))
	g.POST("/resume/save", ginx.BS(h.ResumeSave))
	g.POST("/resume/publish", ginx.BS(h.ResumePublish))
	g.POST("/question/save", ginx.BS(h.QuestionSave))
	g.POST("/question/publish", ginx.BS(h.QuestionPublish))
}

func (h *AdminHandler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	return ginx.Result{
		Data: ProjectList{
			Projects: []Project{
				h.mockProject(1),
				h.mockProject(2),
				h.mockProject(3),
			},
			Total: 3,
		},
	}, nil
}

func (h *AdminHandler) mockProject(id int64) Project {
	return Project{
		Id:     id,
		Title:  fmt.Sprintf("面试项目- %d", id),
		Desc:   fmt.Sprintf("这是面试项目 - %d", id),
		Status: uint8(id%2 + 1),
		Utime:  time.Now().UnixMilli(),
		Difficulties: []Difficulty{
			h.mockDifficult(1),
			h.mockDifficult(2),
			h.mockDifficult(3),
			h.mockDifficult(4),
		},
	}
}

func (h *AdminHandler) mockDifficult(id int64) Difficulty {
	return Difficulty{
		Id:       id,
		Title:    fmt.Sprintf("项目难点- %d", id),
		Analysis: fmt.Sprintf("这是项目难点 - %d", id),
		Content:  fmt.Sprintf("这是面试时候的话术 - %d", id),
		Status:   uint8(id%2 + 1),
		Utime:    time.Now().UnixMilli(),
	}
}

func (h *AdminHandler) Detail(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	return ginx.Result{
		Data: h.mockProject(123),
	}, nil
}

func (h *AdminHandler) DifficultySave(ctx *ginx.Context,
	req DifficultySaveReq, sess session.Session) (ginx.Result, error) {
	return ginx.Result{
		// 返回 Difficulty 的 id
		Data: 123,
	}, nil
}

func (h *AdminHandler) DifficultyPublish(ctx *ginx.Context,
	req DifficultySaveReq, sess session.Session) (ginx.Result, error) {
	return ginx.Result{
		// 返回 Difficulty 的 id
		Data: 123,
	}, nil
}

func (h *AdminHandler) ResumeSave(ctx *ginx.Context,
	req ResumeSaveReq, sess session.Session) (ginx.Result, error) {
	return ginx.Result{
		// 返回 Difficulty 的 id
		Data: 123,
	}, nil
}

func (h *AdminHandler) ResumePublish(ctx *ginx.Context,
	req ResumeSaveReq, sess session.Session) (ginx.Result, error) {
	return ginx.Result{
		// 返回 Difficulty 的 id
		Data: 123,
	}, nil
}

func (h *AdminHandler) QuestionSave(ctx *ginx.Context,
	req QuestionSaveReq, sess session.Session) (ginx.Result, error) {
	return ginx.Result{
		// 返回 Difficulty 的 id
		Data: 123,
	}, nil
}

func (h *AdminHandler) QuestionPublish(ctx *ginx.Context,
	req QuestionSaveReq, sess session.Session) (ginx.Result, error) {
	return ginx.Result{
		// 返回 Difficulty 的 id
		Data: 123,
	}, nil
}

func (h *AdminHandler) Save(ctx *ginx.Context, req Project) (ginx.Result, error) {
	return ginx.Result{
		Data: 123,
	}, nil
}

func (h *AdminHandler) Publish(ctx *ginx.Context, req Project) (ginx.Result, error) {
	return ginx.Result{
		Data: 123,
	}, nil
}
