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
	"github.com/ecodeclub/webook/internal/interview/internal/domain"
	"github.com/ecodeclub/webook/internal/interview/internal/service"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &InterviewJourneyHandler{}

// InterviewJourneyHandler 负责处理面试历程相关的HTTP请求
type InterviewJourneyHandler struct {
	svc service.InterviewService
}

func NewInterviewJourneyHandler(svc service.InterviewService) *InterviewJourneyHandler {
	return &InterviewJourneyHandler{svc: svc}
}

// PrivateRoutes 注册需要登录才能访问的路由
func (h *InterviewJourneyHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/interview-journeys")
	g.POST("/save", ginx.BS[SaveReq](h.Save))
	g.POST("/list", ginx.BS[ListReq](h.List))
	g.POST("/detail", ginx.BS[DetailReq](h.Detail))
}

func (h *InterviewJourneyHandler) PublicRoutes(_ *gin.Engine) {}

// Save 创建一个新的面试历程
func (h *InterviewJourneyHandler) Save(ctx *ginx.Context, req SaveReq, sess session.Session) (ginx.Result, error) {
	id, err := h.svc.Save(ctx, h.toDomain(sess.Claims().Uid, req.Journey))
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *InterviewJourneyHandler) toDomain(uid int64, journey Journey) domain.InterviewJourney {
	return domain.InterviewJourney{
		ID:          journey.ID,
		Uid:         uid,
		CompanyID:   journey.CompanyID,
		CompanyName: journey.CompanyName,
		JobInfo:     journey.JobInfo,
		ResumeURL:   journey.ResumeURL,
		Status:      domain.JourneyStatus(journey.Status),
		Stime:       journey.Stime,
		Etime:       journey.Etime,
		Rounds: slice.Map(journey.Rounds, func(_ int, src Round) domain.InterviewRound {
			return domain.InterviewRound{
				ID:            src.ID,
				Uid:           uid,
				RoundNumber:   src.RoundNumber,
				RoundType:     src.RoundType,
				InterviewDate: src.InterviewDate,
				JobInfo:       src.JobInfo,
				ResumeURL:     src.ResumeURL,
				AudioURL:      src.AudioURL,
				SelfResult:    src.SelfResult,
				SelfSummary:   src.SelfSummary,
				Result:        domain.RoundResult(src.Result),
				AllowSharing:  src.AllowSharing,
			}
		}),
	}
}

// List 获取当前用户的所有面试历程
func (h *InterviewJourneyHandler) List(ctx *ginx.Context, req ListReq, sess session.Session) (ginx.Result, error) {
	journeys, total, err := h.svc.List(ctx, sess.Claims().Uid, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: ginx.DataList[Journey]{
			List: slice.Map(journeys, func(idx int, src domain.InterviewJourney) Journey {
				return h.toVO(src)
			}),
			Total: int(total),
		},
	}, nil
}

func (h *InterviewJourneyHandler) toVO(j domain.InterviewJourney) Journey {
	return Journey{
		ID:          j.ID,
		CompanyID:   j.CompanyID,
		CompanyName: j.CompanyName,
		JobInfo:     j.JobInfo,
		ResumeURL:   j.ResumeURL,
		Status:      j.Status.String(),
		Stime:       j.Stime,
		Etime:       j.Etime,
		Rounds: slice.Map(j.Rounds, func(_ int, src domain.InterviewRound) Round {
			return Round{
				ID:            src.ID,
				RoundNumber:   src.RoundNumber,
				RoundType:     src.RoundType,
				InterviewDate: src.InterviewDate,
				JobInfo:       src.JobInfo,
				ResumeURL:     src.ResumeURL,
				AudioURL:      src.AudioURL,
				SelfResult:    src.SelfResult,
				SelfSummary:   src.SelfSummary,
				Result:        src.Result.String(),
				AllowSharing:  src.AllowSharing,
			}
		}),
	}
}

// Detail 获取面试历程的完整信息
func (h *InterviewJourneyHandler) Detail(ctx *ginx.Context, req DetailReq, sess session.Session) (ginx.Result, error) {
	j, err := h.svc.Detail(ctx, req.ID, sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toVO(j),
	}, nil
}
