package web

import (
	"net/http"
	"time"

	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/service"
)

const (
	//密码规则：长度至少 6 位
	passwordRegexPattern = `^.{6,}$`
	AccessSecret         = "95osj3fUD7fo0mlYdDbncXz4VD2igvf0"
	RefreshSecret        = "95osj3fUD7fo0m123DbncXz4VD2igvf0"
)

type UserHandler struct {
	svc              service.UserAndService
	passwordRegexExp *regexp.Regexp
}

type TokenClaims struct {
	jwt.RegisteredClaims
	// 这是一个前端采集了用户的登录环境生成的一个码
	Fingerprint string
	//用于查找用户信息的一个字段
	Uid int64
}

func NewUserHandler(svc service.UserAndService) *UserHandler {
	return &UserHandler{
		svc:              svc,
		passwordRegexExp: regexp.MustCompile(passwordRegexPattern, regexp.None),
	}
}

func (u *UserHandler) SignUp(ctx *gin.Context) {
	type UserInfo struct {
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirmPassword"`
	}

	var info UserInfo
	if err := ctx.Bind(&info); err != nil {
		return
	}

	//密码和确认密码
	if info.Password != info.ConfirmPassword {
		ctx.String(http.StatusBadRequest, "两次密码不相同！")
		return
	}
	//密码规律
	pwdFlag, err := u.passwordRegexExp.MatchString(info.Password)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统错误！")
		return
	}
	if !pwdFlag {
		ctx.String(http.StatusBadRequest, "密码格式不正确,长度不能小于 6 位！")
		return
	}

	//存储数据...
	err = u.svc.Signup(ctx.Request.Context(), &domain.User{
		Email:    info.Email,
		Password: info.Password,
	})
	if err == service.ErrUserDuplicate {
		ctx.String(http.StatusBadRequest, "重复邮箱，请更换邮箱！")
		return
	}
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统错误！")
		return
	}
	ctx.String(http.StatusOK, "注册成功！")
}

func (u *UserHandler) Login(ctx *gin.Context) {
	type TokenLoginReq struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required"`
		Fingerprint string `json:"fingerprint" binding:"required"` //你可以认为这是一个前端采集了用户的登录环境生成的一个码，你编码进去 EncryptionHandle acccess_token 中。
	}
	var req TokenLoginReq
	err := ctx.ShouldBind(&req)
	if err != nil {
		ctx.String(http.StatusBadRequest, "参数合法性验证失败")
		return
	}

	// 先定义一个uid,实际要在数据库里面查一下
	var uid int64
	err = u.setAccessToken(ctx, req.Fingerprint, uid)
	if err != nil {
		ctx.String(http.StatusBadRequest, "系统错误")
		return
	}

	ctx.String(http.StatusOK, "登陆成功")
}

func (u *UserHandler) setAccessToken(ctx *gin.Context, fingerprint string, uid int64) error {
	now := time.Now()
	//TODO access token
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute * 30)),
		},
		Fingerprint: fingerprint,
		Uid:         uid,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	accessToken, err := token.SignedString([]byte(AccessSecret))
	if err != nil {
		return err
	}
	//TODO refresh token
	claims.RegisteredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(time.Hour * 24 * 7))
	token = jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	refreshToken, err := token.SignedString([]byte(RefreshSecret))
	if err != nil {
		return err
	}

	//TODO 设置token
	ctx.Header("x-access-token", accessToken)
	//可以换一种方式保持到redis里面,避免refresh_token 被人拿到之后一直使用
	//可以使用MD5 转一下,或者直接截取指定长度的字符串 如: 以key 为 前面获取到的字符串
	ctx.Header("x-refresh-token", refreshToken)

	return nil
}
