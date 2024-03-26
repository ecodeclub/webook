package web

import (
	"strconv"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/ecodeclub/webook/internal/user/internal/errs"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/gin-gonic/gin"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	weSvc   service.OAuth2Service
	userSvc service.UserService
	// 白名单
	creators []string
}

func NewHandler(weSvc service.OAuth2Service,
	userSvc service.UserService, creators []string) *Handler {
	return &Handler{
		weSvc:    weSvc,
		userSvc:  userSvc,
		creators: creators,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	users := server.Group("/users")
	users.GET("/profile", ginx.S(h.Profile))
	users.POST("/profile", ginx.BS[EditReq](h.Edit))
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	oauth2 := server.Group("/oauth2")
	oauth2.GET("/wechat/auth_url", ginx.W(h.WechatAuthURL))
	oauth2.Any("/wechat/callback", ginx.B[WechatCallback](h.Callback))
	oauth2.Any("/wechat/token/refresh", ginx.W(h.RefreshAccessToken))
}

func (h *Handler) WechatAuthURL(ctx *ginx.Context) (ginx.Result, error) {
	res, err := h.weSvc.AuthURL()
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
	u, err := h.userSvc.Profile(ctx, sess.Claims().Uid)
	if err != nil {
		return ginx.Result{}, err
	}
	return ginx.Result{
		Data: Profile{
			IsCreator: sess.Claims().Get("creator").StringOrDefault("") == "true",
			Nickname:  u.Nickname,
			Avatar:    u.Avatar,
		},
	}, nil
}

type EditReq struct {
	Avatar   string `json:"avatar"`
	Nickname string `json:"nickname"`
}

// Edit 用户编译信息
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
	info, err := h.weSvc.VerifyCode(ctx, req.Code)
	if err != nil {
		return systemErrorResult, err
	}
	user, err := h.userSvc.FindOrCreateByWechat(ctx, info)
	if err != nil {
		return systemErrorResult, err
	}
	creator := slice.Contains(h.creators, user.WechatInfo.UnionId)
	_, err = session.NewSessionBuilder(ctx, user.Id).
		// 设置是否 creator 的标记位，后续引入权限控制再来改造
		SetJwtData(map[string]string{
			"creator": strconv.FormatBool(creator),
		}).Build()
	if err != nil {
		return systemErrorResult, err
	}
	return ginx.Result{
		Data: Profile{
			Id:        user.Id,
			Nickname:  user.Nickname,
			Avatar:    user.Avatar,
			IsCreator: creator,
		},
	}, nil
}
