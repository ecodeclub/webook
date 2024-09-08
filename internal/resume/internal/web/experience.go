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
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/resume/internal/domain"
	"github.com/ecodeclub/webook/internal/resume/internal/service"
	"github.com/gin-gonic/gin"
)

type ExperienceHandler struct {
	svc service.ExperienceService
}

func NewExperienceHandler(svc service.ExperienceService) *ExperienceHandler {
	return &ExperienceHandler{
		svc: svc,
	}
}

func (h *ExperienceHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/resume/experience")
	g.POST("/save", ginx.BS[Experience](h.Save))
	g.POST("/list", ginx.S(h.List))
	g.POST("/delete", ginx.BS[Experience](h.Delete))
}

// Save 创建或者更新，如果 req 里面有 id 就是更新
func (h *ExperienceHandler) Save(ctx *ginx.Context,
	req Experience,
	sess session.Session) (ginx.Result, error) {
	id, err := h.svc.SaveExperience(ctx, domain.Experience{
		Id:          req.Id,
		Uid:         sess.Claims().Uid,
		Start:       req.Start,
		End:         req.End,
		Title:       req.Title,
		CompanyName: req.CompanyName,
		Location:    req.Location,
		Responsibilities: slice.Map(req.Responsibilities, func(idx int, src Responsibility) domain.Responsibility {
			return domain.Responsibility{
				Type:    src.Type,
				Content: src.Content,
			}

		}),
		Accomplishments: slice.Map(req.Accomplishments, func(idx int, src Accomplishment) domain.Accomplishment {
			return domain.Accomplishment{
				Type:    src.Type,
				Content: src.Content,
			}

		}),
		Skills: req.Skills,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

// List 查询某个人的项目经历
func (h *ExperienceHandler) List(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	// 可以尝试检测 gap，提醒他你需要一个理由,msg 不为空就是有个overlap，提示需要理由
	explist, msg, err := h.svc.List(ctx, sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, err
	}
	resList := make([]Experience, 0, len(explist))
	for _, exp := range explist {
		resList = append(resList, newExperience(exp))
	}
	return ginx.Result{
		Msg:  msg,
		Data: resList,
	}, nil
}

// Delete 删除某段项目经历
func (h *ExperienceHandler) Delete(ctx *ginx.Context, req Experience, sess session.Session) (ginx.Result, error) {
	err := h.svc.Delete(ctx, sess.Claims().Uid, req.Id)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Msg: "success",
	}, nil
}

// MatchJob 这段工作经历和一个 JD 的匹配程度
// 暂时不需要实现
// 输入 JD，输入一个，要求 AI 输出匹配程度， prompt 里面要提示从哪些角度去匹配
func (h *ExperienceHandler) MatchJob(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	panic("implement me")
}

// AIAnalysis AI 对这份工作经历的分析，也就是写得好不好，是否有竞争力
// Experience 输入，prompt 要从哪些角度分析这段工作经历，
func (h *ExperienceHandler) AIAnalysis(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	panic("implement me")
}
