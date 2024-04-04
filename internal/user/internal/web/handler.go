package web

import (
	"context"
	"strconv"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"github.com/ecodeclub/ginx"
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/member"
	"github.com/ecodeclub/webook/internal/user/internal/domain"
	"github.com/ecodeclub/webook/internal/user/internal/errs"
	"github.com/ecodeclub/webook/internal/user/internal/event"
	"github.com/ecodeclub/webook/internal/user/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gotomicro/ego/core/elog"
)

var _ ginx.Handler = &Handler{}

type Handler struct {
	weSvc     service.OAuth2Service
	userSvc   service.UserService
	memberSvc member.Service
	producer  event.Producer
	// 白名单
	creators []string
	logger   *elog.Component
}

func NewHandler(weSvc service.OAuth2Service,
	userSvc service.UserService, memberSvc member.Service, producer event.Producer, creators []string) *Handler {
	return &Handler{
		weSvc:     weSvc,
		userSvc:   userSvc,
		memberSvc: memberSvc,
		producer:  producer,
		creators:  creators,
		logger:    elog.DefaultLogger,
	}
}

func (h *Handler) PrivateRoutes(server *gin.Engine) {
	users := server.Group("/users")
	users.GET("/profile", ginx.S(h.Profile))
	users.POST("/profile", ginx.BS[EditReq](h.Edit))
	users.POST("/member-benefits", ginx.S(h.MemberBenefits))
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
		return systemErrorResult, err
	}
	res := newProfile(u)
	res.IsCreator = sess.Claims().
		Get("creator").
		StringOrDefault("") == "true"
	res.MemberDDL = sess.Claims().
		Get("memberDDL").
		StringOrDefault("")
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

// MemberBenefits 会员权益页,将会员截止日期设置到
func (h *Handler) MemberBenefits(ctx *ginx.Context, sess session.Session) (ginx.Result, error) {
	claims := sess.Claims()
	memberDDL := claims.Get("memberDDL").StringOrDefault("")
	if memberDDL != "" {
		// 已经设置过会员截止日期,至于是否过期需要进一步检查
		return ginx.Result{}, nil
	}

	// 没有会员截止日期, 可能是未购买,可能购买过但未设置
	memberDDL = h.getMemberDDL(ctx.Request.Context(), claims.Uid)
	if memberDDL != "" {
		// 重新生成session并替换
		jwtData := claims.Data
		jwtData["memberDDL"] = memberDDL
		_, err := session.NewSessionBuilder(ctx, claims.Uid).SetJwtData(jwtData).Build()
		if err != nil {
			return systemErrorResult, err
		}
	}

	return ginx.Result{}, nil
}

func (h *Handler) Callback(ctx *ginx.Context, req WechatCallback) (ginx.Result, error) {
	info, err := h.weSvc.VerifyCode(ctx, req.Code)
	if err != nil {
		return systemErrorResult, err
	}
	user, isCreated, err := h.userSvc.FindOrCreateByWechat(ctx, info)
	if err != nil {
		return systemErrorResult, err
	}

	if isCreated {
		// 发送注册成功消息
		evt := event.RegistrationEvent{UserID: user.Id}
		if e := h.producer.ProduceRegistrationEvent(ctx, evt); e != nil {
			h.logger.Error("发送注册成功消息失败",
				elog.FieldErr(e),
				elog.FieldKey("event"),
				elog.FieldValueAny(evt),
			)
		}
		// 等1s尽最大努力让会员模块消费完
		time.Sleep(1 * time.Second)
	}

	// 构建session
	jwtData := map[string]string{}
	// 设置是否 creator 的标记位，后续引入权限控制再来改造
	isCreator := slice.Contains(h.creators, user.WechatInfo.UnionId)
	jwtData["creator"] = strconv.FormatBool(isCreator)
	// 设置会员截止日期
	memberDDL := h.getMemberDDL(ctx.Request.Context(), user.Id)
	jwtData["memberDDL"] = memberDDL

	_, err = session.NewSessionBuilder(ctx, user.Id).SetJwtData(jwtData).Build()
	if err != nil {
		return systemErrorResult, err
	}

	res := newProfile(user)
	res.IsCreator = isCreator
	res.MemberDDL = memberDDL
	return ginx.Result{
		Data: res,
	}, nil
}

func (h *Handler) getMemberDDL(ctx context.Context, userID int64) string {
	mem, err := h.memberSvc.GetMembershipInfo(ctx, userID)
	if err != nil {
		return ""
	}
	return time.Unix(mem.EndAt, 0).Local().Format(time.DateTime)
}
