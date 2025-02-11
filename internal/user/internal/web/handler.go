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
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/ecodeclub/webook/internal/pkg/middleware"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/ecodeclub/webook/internal/user/internal/errs"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
	"golang.org/x/sync/errgroup"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	weSvc         service.OAuth2Service
	weMiniSvc     service.OAuth2Service
	userSvc       service.UserService
	memberSvc     member.Service
	permissionSvc permission.Service
	// 白名单
	creators []string
	logger   *elog.Component
	sp       session.Provider
}

func NewHandler(
	// 微信
	weSvc service.OAuth2Service,
	// 微信小程序
	weMiniSvc service.OAuth2Service,
	userSvc service.UserService,
	memberSvc member.Service,
	permissionSvc permission.Service,
	sp session.Provider,
	creators []string) *Handler {
	return &Handler{
		weSvc:         weSvc,
		weMiniSvc:     weMiniSvc,
		userSvc:       userSvc,
		memberSvc:     memberSvc,
		permissionSvc: permissionSvc,
		creators:      creators,
		logger:        elog.DefaultLogger,
		sp:            sp,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	users := server.Group("/users")
	users.GET("/profile", ginx.S(h.Profile))
	users.POST("/logout", ginx.W(h.Logout))
	users.POST("/profile", ginx.BS[EditReq](h.Edit))
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	appidFunc := middleware.NewCheckAppIdBuilder().Build()
	oauth2 := server.Group("/oauth2")
	oauth2.GET("/wechat/auth_url", appidFunc, ginx.W(h.WechatAuthURL))
	oauth2.GET("/mock/login", appidFunc, ginx.W(h.MockLogin))
	// 扫码登录回调
	oauth2.Any("/wechat/callback", appidFunc, ginx.B[WechatCallback](h.Callback))

	// 小程序登录回调
	oauth2.Any("/wechat/mini/callback", appidFunc, ginx.B[WechatCallback](h.MiniCallback))
	oauth2.Any("/wechat/token/refresh", appidFunc, ginx.W(h.RefreshAccessToken))
}

func (h *Handler) Logout(ctx *ginx.Context) (ginx.Result, error) {
	err := h.sp.Destroy(ctx)
	if err != nil {
		return systemErrorResult, nil
	}
	return ginx.Result{
		Msg: "OK",
	}, nil
}

func (h *Handler) WechatAuthURL(ctx *ginx.Context) (ginx.Result, error) {
	code, _ := ctx.GetQuery("code")
	res, err := h.weSvc.AuthURL(ctx, service.AuthParams{
		InvitationCode: code,
	})
	if err != nil {
		return ginx.Result{
			Code: errs.SystemError.Code,
			Msg:  errs.SystemError.Msg,
		}, err
	}
	return ginx.Result{
		Data: res,
	}, nil
}

func (h *Handler) RefreshAccessToken(ctx *ginx.Context) (ginx.Result, error) {
	err := session.RenewAccessToken(ctx)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{Msg: "OK"}, nil
}

func (h *Handler) Profile(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	var (
		eg errgroup.Group
		u  domain.User
		m  member.Member
	)
	uid := sess.Claims().Uid
	eg.Go(func() error {
		var err error
		u, err = h.userSvc.Profile(ctx, uid)
		return err
	})

	eg.Go(func() error {
		var err error
		m, err = h.memberSvc.GetMembershipInfo(ctx, uid)
		if err != nil {
			h.logger.Error("查找用户的会员信息失败", elog.FieldErr(err))
		}
		return nil
	})

	err := eg.Wait()
	if err != nil {
		return systemErrorResult, err
	}
	res := newProfile(u)
	res.IsCreator = sess.Claims().
		Get("creator").
		StringOrDefault("") == "true"
	res.MemberDDL = m.EndAt
	return ginx.Result{
		Data: res,
	}, nil
}

// Edit 用户编辑信息
func (h *Handler) Edit(ctx *ginx.Context, req EditReq, sess session.Session) (ginx.Result, error) {
	uid := sess.Claims().Uid
	err := h.userSvc.UpdateNonSensitiveInfo(ctx, domain.User{
		Id:       uid,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
	})
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Msg: "OK",
	}, nil
}

func (h *Handler) Callback(ctx *ginx.Context, req WechatCallback) (ginx.Result, error) {

	info, err := h.weSvc.Verify(ctx, service.CallbackParams{
		Code:  req.Code,
		State: req.State,
	})
	if err != nil {
		return systemErrorResult, err
	}
	user, err := h.userSvc.FindOrCreateByWechat(ctx, info)
	if err != nil {
		return systemErrorResult, err
	}

	res, err := h.setupSession(ctx, user)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: res,
	}, nil
}

func (h *Handler) setupSession(ctx *ginx.Context, user domain.User) (Profile, error) {
	// 构建session
	jwtData := map[string]string{}
	// 设置是否 creator 的标记位，后续引入权限控制再来改造
	isCreator := slice.Contains(h.creators, user.WechatInfo.UnionId)
	jwtData["creator"] = strconv.FormatBool(isCreator)
	// 设置会员截止日期
	memberDDL := h.getMemberDDL(ctx, user.Id)
	jwtData["memberDDL"] = strconv.FormatInt(memberDDL, 10)

	perms := make(map[string]string)
	permissionGroup, err := h.permissionSvc.FindPersonalPermissions(ctx, user.Id)
	if err != nil {
		return Profile{}, err
	}
	for biz, permissions := range permissionGroup {
		bizIds := slice.Map(permissions, func(idx int, src permission.Permission) string {
			return strconv.FormatInt(src.BizID, 10)
		})
		perms[biz] = strings.Join(bizIds, ",")
	}
	permsVal, _ := json.Marshal(perms)
	// redis 不能处理 map[string]map[string][string] 这种二层结构
	sessData := map[string]any{"permission": permsVal}
	_, err = session.NewSessionBuilder(ctx, user.Id).SetJwtData(jwtData).SetSessData(sessData).Build()
	if err != nil {
		return Profile{}, err
	}
	res := newProfile(user)
	res.IsCreator = isCreator
	res.MemberDDL = memberDDL
	return res, nil
}

func (h *Handler) getMemberDDL(ctx context.Context, userID int64) int64 {
	mem, err := h.memberSvc.GetMembershipInfo(ctx, userID)
	if err != nil {
		h.logger.Error("查找会员信息失败", elog.FieldErr(err), elog.Int64("uid", userID))
	}
	return mem.EndAt
}

// MiniCallback 微信小程序登录回调
func (h *Handler) MiniCallback(ctx *ginx.Context, req WechatCallback) (ginx.Result, error) {
	info, err := h.weMiniSvc.Verify(ctx, service.CallbackParams{
		Code:  req.Code,
		State: req.State,
	})
	if err != nil {
		return systemErrorResult, err
	}
	user, err := h.userSvc.FindOrCreateByWechat(ctx, info)
	if err != nil {
		return systemErrorResult, err
	}
	res, err := h.setupSession(ctx, user)
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: res,
	}, nil
}
