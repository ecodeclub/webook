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

	"github.com/ecodeclub/ekit/slice"
	"github.com/gotomicro/ego/core/elog"

	"github.com/ecodeclub/ekit/bean/copier"
	"github.com/ecodeclub/ekit/bean/copier/converter"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	vo2dm  copier.Copier[Question, domain.Question]
	dm2vo  copier.Copier[domain.Question, Question]
	svc    service.Service
	logger *elog.Component
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
		vo2dm:  vo2dm,
		dm2vo:  dm2vo,
		svc:    svc,
		logger: elog.DefaultLogger,
	}, nil
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	server.POST("/question/save", ginx.BS[SaveReq](h.Save))
	server.POST("/question/list", ginx.BS[Page](h.List))
	server.POST("/question/detail", ginx.BS[Qid](h.Detail))
	server.POST("/question/publish", ginx.BS[SaveReq](h.Publish))
	server.POST("/question/pub/list", ginx.B[Page](h.PubList))
	server.POST("/question/pub/detail", ginx.B[Qid](h.PubDetail))
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

func (h *Handler) Publish(ctx *ginx.Context, req SaveReq, sess session.Session) (ginx.Result, error) {
	que, err := h.vo2dm.Copy(&req.Question)
	if err != nil {
		return systemErrorResult, err
	}
	que.Uid = sess.Claims().Uid
	id, err := h.svc.Publish(ctx, que)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: id,
	}, nil
}

func (h *Handler) List(ctx *ginx.Context, req Page, sess session.Session) (ginx.Result, error) {
	// 制作库不需要统计总数
	data, cnt, err := h.svc.List(ctx, req.Offset, req.Limit, sess.Claims().Uid)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionList(data, cnt),
	}, nil
}

func (h *Handler) PubList(ctx *ginx.Context, req Page) (ginx.Result, error) {
	data, cnt, err := h.svc.PubList(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: h.toQuestionList(data, cnt),
	}, nil
}

func (h *Handler) toQuestionList(data []domain.Question, cnt int64) QuestionList {
	return QuestionList{
		Total: cnt,
		Questions: slice.Map(data, func(idx int, src domain.Question) Question {
			// 忽略了错误
			// 在 PubList 里面，不需要答案
			dst, err := h.dm2vo.Copy(&src,
				copier.IgnoreFields("Answer"))
			if err != nil {
				h.logger.Error("转化为 vo 失败", elog.FieldErr(err))
				return Question{}
			}
			return *dst
		}),
	}
}

func (h *Handler) Detail(ctx *ginx.Context, req Qid, sess session.Session) (ginx.Result, error) {
	detail, err := h.svc.Detail(ctx, req.Qid)
	if err != nil {
		return systemErrorResult, err
	}
	if detail.Uid != sess.Claims().Uid {
		// 非法访问，说明有人搞鬼
		// 在有人搞鬼的时候，直接返回系统错误就可以
		return systemErrorResult, err
	}
	vo, err := h.dm2vo.Copy(&detail)
	return ginx.Result{
		Data: vo,
	}, err
}

func (h *Handler) PubDetail(ctx *ginx.Context, req Qid) (ginx.Result, error) {
	detail, err := h.svc.PubDetail(ctx, req.Qid)
	if err != nil {
		return systemErrorResult, err
	}
	vo, err := h.dm2vo.Copy(&detail)
	return ginx.Result{
		Data: vo,
	}, err
}
