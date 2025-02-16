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
	"strconv"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/pkg/html_truncate"
	"github.com/ecodeclub/webook/internal/question/internal/domain"
	"github.com/ecodeclub/webook/internal/question/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/sync/errgroup"
)

type Handler struct {
	logger     *elog.Component
	intrSvc    interactive.Service
	examineSvc service.ExamineService
	svc        service.Service
	permSvc    permission.Service
	// truncator 进行html的裁剪
	truncator html_truncate.HTMLTruncator
	sp        session.Provider
	memberSvc member.Service
}

func NewHandler(intrSvc interactive.Service,
	examineSvc service.ExamineService,
	permSvc permission.Service,
	svc service.Service,
	sp session.Provider,
	memberSvc member.Service,
) *Handler {
	return &Handler{
		intrSvc:    intrSvc,
		permSvc:    permSvc,
		examineSvc: examineSvc,
		svc:        svc,
		memberSvc:  memberSvc,
		sp:         sp,
		truncator:  html_truncate.DefaultHTMLTruncator(),
	}
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	server.POST("/question/list", ginx.B[Page](h.PubList))
	server.POST("/question/detail", ginx.B[Qid](h.PubDetail))
}

func (h *Handler) PubDetail(ctx *ginx.Context,
	req Qid) (ginx.Result, error) {
	var (
		eg      errgroup.Group
		intr    interactive.Interactive
		examine domain.Result
	)

	detail, err := h.svc.PubDetail(ctx, req.Qid)
	if err != nil {
		return systemErrorResult, fmt.Errorf("查找面试题详情失败 %w", err)
	}
	has, uid := h.checkPermission(ctx, detail)
	// 没权限就返回部分数据
	if !has {
		detail = h.partQuestion(detail)
	}
	// 非八股文，我们需要判定是否有权限
	// 暂时在这里聚合
	eg.Go(func() error {
		// uid 可能为 0，在为 0 的时候多查询了用户本身是否已经点赞收藏过的信息
		// 后续要优化
		var err error
		intr, err = h.intrSvc.Get(ctx, domain.QuestionBiz, req.Qid, uid)
		return err
	})

	eg.Go(func() error {
		var err error
		// uid 为 0 的时候，肯定没测试结果，后续要优化
		examine, err = h.examineSvc.QuestionResult(ctx, uid, req.Qid)
		return err
	})
	err = eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}

	que := newQuestion(detail, intr)
	que.ExamineResult = examine.ToUint8()
	// 记录是否有权限
	que.Permitted = has
	return ginx.Result{
		Data: que,
	}, err
}
func (h *Handler) getUid(gctx *ginx.Context) int64 {
	sess, err := h.sp.Get(gctx)
	if err != nil {
		// 没登录
		return 0
	}
	return sess.Claims().Uid
}
func (h *Handler) PubList(ctx *ginx.Context, req Page) (ginx.Result, error) {
	uid := h.getUid(ctx)
	count, data, err := h.svc.PubList(ctx, req.Offset, req.Limit)
	if err != nil {
		return systemErrorResult, err
	}
	// 查询点赞收藏记录
	intrs := map[int64]interactive.Interactive{}
	if len(data) > 0 {
		ids := slice.Map(data, func(idx int, src domain.Question) int64 {
			return src.Id
		})
		var err1 error
		intrs, err1 = h.intrSvc.GetByIds(ctx, "question", uid, ids)
		// 这个数据查询不到也不需要担心
		if err1 != nil {
			h.logger.Error("查询数据的点赞数据失败",
				elog.Any("ids", ids),
				elog.FieldErr(err))
		}
	}

	// 获得数据
	return ginx.Result{
		Data: h.toQuestionList(data, count, intrs),
	}, nil
}

func (h *Handler) toQuestionList(data []domain.Question, cnt int64, intrs map[int64]interactive.Interactive) QuestionList {
	return QuestionList{
		Total: cnt,
		Questions: slice.Map(data, func(idx int, src domain.Question) Question {
			return newQuestion(src, intrs[src.Id])
		}),
	}
}

func (h *Handler) partQuestion(que domain.Question) domain.Question {
	que.Answer.Analysis.Content = h.truncator.Truncate(que.Answer.Analysis.Content)
	que.Answer.Advanced.Content = h.truncator.Truncate(que.Answer.Advanced.Content)
	que.Answer.Intermediate.Content = h.truncator.Truncate(que.Answer.Intermediate.Content)
	que.Answer.Basic.Content = h.truncator.Truncate(que.Answer.Basic.Content)
	return que
}

func (h *Handler) checkPermission(gctx *ginx.Context, que domain.Question) (bool, int64) {
	sess, err := h.sp.Get(gctx)
	if err != nil {
		// 没登录
		return false, 0
	}
	uid := sess.Claims().Uid
	// 是八股文校验是否是会员
	if que.IsBaguwen() {
		claims := sess.Claims()
		memberDDL, _ := claims.Get("memberDDL").AsInt64()
		// 如果 jwt 中的数据格式不对，那么这里就会返回 0
		// jwt中找到会员截止日期，没有过期
		if memberDDL > time.Now().UnixMilli() {
			return true, uid
		}
		info, err := h.memberSvc.GetMembershipInfo(gctx, uid)
		if err != nil {
			return false, uid
		}
		if info.EndAt == 0 {
			return false, uid
		}
		if info.EndAt < time.Now().UnixMilli() {
			return false, uid
		}
		// 在原有jwt数据中添加会员截止日期
		jwtData := claims.Data
		jwtData["memberDDL"] = strconv.FormatInt(info.EndAt, 10)
		claims.Data = jwtData
		err = h.sp.UpdateClaims(gctx, claims)
		if err != nil {
			elog.Error("重新生成 token 失败", elog.Int64("uid", claims.Uid), elog.FieldErr(err))
			return true, uid
		}
		return true, uid
	} else {
		var ok bool
		ok, err = h.permSvc.HasPermission(gctx, permission.Permission{
			Uid:   uid,
			Biz:   que.Biz,
			BizID: que.BizId,
		})
		if err != nil {
			return false, uid
		}
		if !ok {
			return false, uid
		}
		return true, uid
	}
}
