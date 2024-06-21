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
	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/project/internal/domain"
	"github.com/ecodeclub/webook/internal/project/internal/service"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

type AdminHandler struct {
	svc service.ProjectAdminService
}

func (h *AdminHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/project")
	g.POST("/list", ginx.B[Page](h.List))
	g.POST("/detail", ginx.B[IdReq](h.Detail))
	g.POST("/save", ginx.B[Project](h.Save))
	g.POST("/publish", ginx.B[Project](h.Publish))
	g.POST("/delete", ginx.B[IdReq](h.Delete))
	// 上架，也就是作为一个面试项目发布出去

	g.POST("/difficulty/save", ginx.B(h.DifficultySave))
	g.POST("/difficulty/detail", ginx.B(h.DifficultyDetail))
	g.POST("/difficulty/publish", ginx.B(h.DifficultyPublish))

	g.POST("/resume/save", ginx.B(h.ResumeSave))
	g.POST("/resume/publish", ginx.B(h.ResumePublish))
	g.POST("/resume/detail", ginx.B(h.ResumeDetail))

	g.POST("/question/save", ginx.B(h.QuestionSave))
	g.POST("/question/detail", ginx.B(h.QuestionDetail))
	g.POST("/question/publish", ginx.B(h.QuestionPublish))

	g.POST("/introduction/save", ginx.B(h.IntroductionSave))
	g.POST("/introduction/detail", ginx.B(h.IntroductionDetail))
	g.POST("/introduction/publish", ginx.B(h.IntroductionPublish))

	// 面试小套路，连招
	g.POST("/combo/save", ginx.B(h.ComboSave))
	g.POST("/combo/detail", ginx.B(h.ComboDetail))
	g.POST("/combo/publish", ginx.B(h.ComboPublish))
}

func (h *AdminHandler) List(ctx *ginx.Context, req Page) (ginx.Result, error) {
	var (
		eg   errgroup.Group
		list []domain.Project
		cnt  int64
	)

	eg.Go(func() error {
		var err error
		list, err = h.svc.List(ctx, req.Offset, req.Limit)
		return err
	})
	eg.Go(func() error {
		var err error
		cnt, err = h.svc.Count(ctx)
		return err
	})
	if err := eg.Wait(); err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: ProjectList{
			Projects: slice.Map(list, func(idx int, src domain.Project) Project {
				return newProject(src, interactive.Interactive{})
			}),
			Total: cnt,
		},
	}, nil
}

func (h *AdminHandler) Detail(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	prj, err := h.svc.Detail(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newProject(prj, interactive.Interactive{}),
	}, nil
}

func (h *AdminHandler) DifficultySave(ctx *ginx.Context,
	req DifficultySaveReq) (ginx.Result, error) {
	id, err := h.svc.DifficultySave(ctx, req.Pid, req.Difficulty.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		// 返回 Difficulty 的 id
		Data: id,
	}, nil
}

func (h *AdminHandler) DifficultyPublish(ctx *ginx.Context,
	req DifficultySaveReq) (ginx.Result, error) {
	id, err := h.svc.DifficultyPublish(ctx, req.Pid, req.Difficulty.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) ResumeSave(ctx *ginx.Context,
	req ResumeSaveReq) (ginx.Result, error) {
	id, err := h.svc.ResumeSave(ctx, req.Pid, req.Resume.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) ResumePublish(ctx *ginx.Context,
	req ResumeSaveReq) (ginx.Result, error) {
	id, err := h.svc.ResumePublish(ctx, req.Pid, req.Resume.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		// 返回 Difficulty 的 id
		Data: id,
	}, nil
}

func (h *AdminHandler) QuestionSave(ctx *ginx.Context,
	req QuestionSaveReq) (ginx.Result, error) {
	id, err := h.svc.QuestionSave(ctx, req.Pid, req.Question.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) QuestionPublish(ctx *ginx.Context,
	req QuestionSaveReq) (ginx.Result, error) {
	id, err := h.svc.QuestionPublish(ctx, req.Pid, req.Question.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) Save(ctx *ginx.Context, req Project) (ginx.Result, error) {
	id, err := h.svc.Save(ctx, req.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) Publish(ctx *ginx.Context, req Project) (ginx.Result, error) {
	id, err := h.svc.Publish(ctx, req.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) ResumeDetail(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	res, err := h.svc.ResumeDetail(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newResume(res),
	}, nil
}

func (h *AdminHandler) DifficultyDetail(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	res, err := h.svc.DifficultyDetail(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newDifficulty(res),
	}, nil
}

func (h *AdminHandler) QuestionDetail(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	res, err := h.svc.QuestionDetail(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newQuestion(res),
	}, nil
}

func (h *AdminHandler) IntroductionSave(ctx *ginx.Context,
	req IntroductionSaveReq) (ginx.Result, error) {
	id, err := h.svc.IntroductionSave(ctx, req.Pid, req.Introduction.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{Data: id}, nil
}

func (h *AdminHandler) IntroductionDetail(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	res, err := h.svc.IntroductionDetail(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newIntroduction(res),
	}, nil
}

func (h *AdminHandler) IntroductionPublish(ctx *ginx.Context,
	req IntroductionSaveReq) (ginx.Result, error) {
	id, err := h.svc.IntroductionPublish(ctx, req.Pid, req.Introduction.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) ComboSave(ctx *ginx.Context, req ComboSaveReq) (ginx.Result, error) {
	id, err := h.svc.ComboSave(ctx, req.Pid, req.Combo.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) ComboDetail(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	res, err := h.svc.ComboDetail(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: newCombo(res),
	}, nil
}

func (h *AdminHandler) ComboPublish(ctx *ginx.Context, req ComboSaveReq) (ginx.Result, error) {
	id, err := h.svc.ComboPublish(ctx, req.Pid, req.Combo.toDomain())
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *AdminHandler) Delete(ctx *ginx.Context, req IdReq) (ginx.Result, error) {
	err := h.svc.Delete(ctx, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{Msg: "OK"}, nil
}

func NewAdminHandler(svc service.ProjectAdminService) *AdminHandler {
	return &AdminHandler{
		svc: svc,
	}
}
