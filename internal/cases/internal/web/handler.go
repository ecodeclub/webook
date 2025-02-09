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
	"strconv"
	"time"

	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/pkg/html_truncate"
	"golang.org/x/sync/errgroup"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/cases/internal/domain"
	"github.com/ecodeclub/webook/internal/cases/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	svc        service.Service
	intrSvc    interactive.Service
	examineSvc service.ExamineService
	logger     *elog.Component

	truncator html_truncate.HTMLTruncator
	sp        session.Provider
	memberSvc member.Service
}

func NewHandler(svc service.Service,
	examineSvc service.ExamineService,
	intrSvc interactive.Service,
	memberSvc member.Service,
	sp session.Provider,
) *Handler {
	return &Handler{
		svc:        svc,
		intrSvc:    intrSvc,
		examineSvc: examineSvc,
		logger:     elog.DefaultLogger,
		memberSvc:  memberSvc,
		sp:         sp,
		truncator:  html_truncate.DefaultHTMLTruncator(),
	}
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	server.POST("/case/list", ginx.B[Page](h.PubList))
	server.POST("/case/detail", ginx.B(h.PubDetail))
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

	intrs := map[int64]interactive.Interactive{}
	if len(data) > 0 {
		ids := slice.Map(data, func(idx int, src domain.Case) int64 {
			return src.Id
		})
		var err1 error
		intrs, err1 = h.intrSvc.GetByIds(ctx, "case", uid, ids)
		// 这个数据查询不到也不需要担心
		if err1 != nil {
			h.logger.Error("查询数据的点赞数据失败",
				elog.Any("ids", ids),
				elog.FieldErr(err))
		}
	}
	return ginx.Result{
		Data: CasesList{
			Total: count,
			Cases: slice.Map(data, func(idx int, ca domain.Case) Case {
				return Case{
					Id:           ca.Id,
					Title:        ca.Title,
					Introduction: ca.Introduction,
					Labels:       ca.Labels,
					Utime:        ca.Utime.UnixMilli(),
					Interactive:  newInteractive(intrs[ca.Id]),
				}
			}),
		},
	}, nil
}

func (h *Handler) PubDetail(ctx *ginx.Context, req CaseId) (ginx.Result, error) {
	var (
		eg         errgroup.Group
		detail     domain.Case
		intr       interactive.Interactive
		exmaineRes domain.CaseResult
	)

	var err error
	detail, err = h.svc.PubDetail(ctx, req.Cid)
	if err != nil {
		return systemErrorResult, err
	}
	has, uid := h.checkPermission(ctx)
	if !has {
		detail = h.partCase(detail)
	}
	eg.Go(func() error {
		var err error
		intr, err = h.intrSvc.Get(ctx, domain.BizCase, req.Cid, uid)
		return err
	})

	eg.Go(func() error {
		var err error
		exmaineRes, err = h.examineSvc.GetResult(ctx, uid, req.Cid)
		return err
	})

	err = eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}
	res := newCase(detail)
	res.Interactive = newInteractive(intr)
	res.ExamineResult = exmaineRes.ToUint8()
	res.Permitted = has
	return ginx.Result{
		Data: res,
	}, err
}

func newCase(ca domain.Case) Case {
	return Case{
		Id:           ca.Id,
		Title:        ca.Title,
		Introduction: ca.Introduction,
		Content:      ca.Content,
		Labels:       ca.Labels,
		GiteeRepo:    ca.GiteeRepo,
		GithubRepo:   ca.GithubRepo,
		Keywords:     ca.Keywords,
		Shorthand:    ca.Shorthand,
		Highlight:    ca.Highlight,
		Guidance:     ca.Guidance,
		Biz:          ca.Biz,
		BizId:        ca.BizId,
		Status:       ca.Status.ToUint8(),
		Utime:        ca.Utime.UnixMilli(),
	}
}

func (h *Handler) partCase(ca domain.Case) domain.Case {
	ca.Content = h.truncator.Truncate(ca.Content)
	return ca
}

func (h *Handler) checkPermission(gctx *ginx.Context) (bool, int64) {
	sess, err := h.sp.Get(gctx)
	if err != nil {
		// 没登录
		return false, 0
	}
	uid := sess.Claims().Uid
	// 是八股文校验是否是会员

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
}
