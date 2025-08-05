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
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interview/internal/domain"
	"github.com/ecodeclub/webook/internal/interview/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var _ ginx.Handler = &InterviewRoundHandler{}

// InterviewRoundHandler 负责处理面试轮次相关的HTTP请求
type InterviewRoundHandler struct {
	svc service.InterviewRoundService
}

func NewInterviewRoundHandler(svc service.InterviewRoundService) *InterviewRoundHandler {
	return &InterviewRoundHandler{svc: svc}
}

func (h *InterviewRoundHandler) PrivateRoutes(server *gin.Engine) {
	g := server.Group("/interview-rounds")
	g.POST("/create", ginx.BS[CreateRoundReq](h.Create))
	g.POST("/update", ginx.BS[UpdateRoundReq](h.Update))
}

func (h *InterviewRoundHandler) PublicRoutes(_ *gin.Engine) {}

// Create 为一个面试历程添加新的轮次
func (h *InterviewRoundHandler) Create(ctx *ginx.Context, req CreateRoundReq, sess session.Session) (ginx.Result, error) {
	req.Round.ID = 0
	round := h.toDomain(sess.Claims().Uid, req.Round)
	if !round.Result.IsValid() {
		return systemErrorResult, errors.New("官方结果非法")
	}
	id, err := h.svc.Create(ctx, round)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *InterviewRoundHandler) toDomain(uid int64, round Round) domain.InterviewRound {
	return domain.InterviewRound{
		ID:            round.ID,
		Jid:           round.Jid,
		Uid:           uid,
		RoundNumber:   round.RoundNumber,
		RoundType:     round.RoundType,
		InterviewDate: round.InterviewDate,
		JobInfo:       round.JobInfo,
		ResumeURL:     round.ResumeURL,
		AudioURL:      round.AudioURL,
		SelfResult:    round.SelfResult,
		SelfSummary:   round.SelfSummary,
		Result:        domain.RoundResult(round.Result),
		AllowSharing:  round.AllowSharing,
	}
}

// Update 更新一个面试轮次的信息
func (h *InterviewRoundHandler) Update(ctx *ginx.Context, req UpdateRoundReq, sess session.Session) (ginx.Result, error) {
	err := h.svc.Update(ctx, h.toDomain(sess.Claims().Uid, req.Round))
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{Msg: "OK"}, nil
}
