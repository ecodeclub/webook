package web

import (
	"net/http"

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
}

func NewHandler(weSvc service.OAuth2Service,
	userSvc service.UserService) *Handler {
	return &Handler{
		weSvc:   weSvc,
		userSvc: userSvc,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	users := server.Group("/users")
	users.GET("/profile", ginx.S(h.Profile))
	users.POST("/profile", ginx.BS[EditReq](h.Edit))
	users.GET("/401", func(ctx *gin.Context) {
		ctx.String(http.StatusUnauthorized, "test")
	})
}

func (h *Handler) PublicRoutes(server *gin.Engine) {
	oauth2 := server.Group("/oauth2")
	oauth2.GET("/wechat/auth_url", ginx.W(h.WechatAuthURL))
	oauth2.Any("/wechat/callback", ginx.W(h.Callback))
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
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
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

func (h *Handler) Callback(ctx *ginx.Context) (ginx.Result, error) {
	info, err := h.weSvc.VerifyCode(ctx, ctx.Query("code").StringOrDefault(""))
	if err != nil {
		return systemErrorResult, err
	}
	user, err := h.userSvc.FindOrCreateByWechat(ctx, info)
	if err != nil {
		return systemErrorResult, err
	}
	_, err = session.NewSessionBuilder(ctx, user.Id).Build()
	if err != nil {
		return systemErrorResult, err
	}
	// 固定跳转，后续考虑灵活跳转
	ctx.Redirect(http.StatusTemporaryRedirect, "/")
	return ginx.Result{
		Data: Profile{
			Id:       user.Id,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		},
	}, nil
}
