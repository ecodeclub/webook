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

	"github.com/ecodeclub/ekit/bean/copier"
	"github.com/ecodeclub/ekit/bean/copier/converter"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	vo2dm copier.Copier[Question, domain.Question]
	dm2vo copier.Copier[domain.Question, Question]
	svc   service.Service
}

func NewHandler(svc service.Service) (*Handler, error) {
	vo2dm, err := copier.NewReflectCopier[Question, domain.Question](
		copier.IgnoreFields("Utime"),
	)
	if err != nil {
		return nil, err
	}
	cnvter := converter.ConverterFunc[time.Time, string](func(src time.Time) (string, error) {
		return src.Format(time.DateTime), nil
	})
	dm2vo, err := copier.NewReflectCopier[domain.Question, Question](
		copier.ConvertField[time.Time, string]("Utime", cnvter),
	)
	if err != nil {
		return nil, err
	}
	return &Handler{
		vo2dm: vo2dm,
		dm2vo: dm2vo,
		svc:   svc,
	}, nil
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/question", ginx.BS[SaveReq](h.Save))
}

func (h *Handler) PublicRoutes(server *gin.Engine) {}

func (h *Handler) Save(ctx *ginx.Context,
	req SaveReq,
	sess session.Session) (ginx.Result, error) {
	que, err := h.vo2dm.Copy(&req.Question)
	if err != nil {
		return systemErrorResult, err
	}
	que.Uid = sess.Claims().Uid
	id, err := h.svc.Save(ctx, que)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}
