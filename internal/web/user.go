package web

import (
	"net/http"
	"time"

	"go.uber.org/zap"

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
	birthdayRegexPatten  = `^(19|20)\d\d[-](0[1-9]|1[012])[-](0[1-9]|[12][0-9]|3[01])$`
	aboutMeMaxLen        = 1024
	nickNameMaxLen       = 128
)

type UserHandler struct {
	svc                 service.UserService
	passwordRegexExp    *regexp.Regexp
	birthdayRegexPatten *regexp.Regexp
}

type TokenClaims struct {
	jwt.RegisteredClaims
	// 这是一个前端采集了用户的登录环境生成的一个码
	Fingerprint string
	//用于查找用户信息的一个字段
	Uid int64
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{
		svc:                 svc,
		passwordRegexExp:    regexp.MustCompile(passwordRegexPattern, regexp.None),
		birthdayRegexPatten: regexp.MustCompile(birthdayRegexPatten, regexp.None),
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

	//发送验证邮箱的邮件
	err = u.svc.SendVerifyEmail(ctx.Request.Context(), info.Email)
	if err != nil {
		//TODO： 用zap写日志
		return
	}
}

func (u *UserHandler) EmailVerify(ctx *gin.Context) {
	token := ctx.Param("token")

	err := u.svc.VerifyEmail(ctx, token)
	if err != nil {
		ctx.String(http.StatusOK, "验证失败!")
		return
	}
	ctx.String(http.StatusOK, "验证成功!")
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
	ctx.Set("userid", uid)

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

// Edit 用户编译信息
func (c *UserHandler) Edit(ctx *gin.Context) {
	type UserEditReq struct {
		Id       int64
		NickName string
		Birthday string
		AboutMe  string
	}
	var req UserEditReq

	if err := ctx.Bind(&req); err != nil {
		return
	}

	if len(req.NickName) > nickNameMaxLen {
		ctx.String(http.StatusOK, "昵称超过长度限制！")
		return
	}

	//校验生日格式是否合法
	isBirthday, err := c.birthdayRegexPatten.MatchString(req.Birthday)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	if !isBirthday {
		ctx.String(http.StatusOK,
			"非法的生日日期，标准样式为：yyyy-mm-dd")
		return
	}
	//校验简介长度
	if len(req.AboutMe) > aboutMeMaxLen {
		ctx.String(http.StatusOK, "简介超过长度限制！")
		return
	}
	err = c.svc.EditUserProfile(ctx, domain.User{
		Id:       req.Id,
		Birthday: req.Birthday,
		NickName: req.NickName,
		AboutMe:  req.AboutMe,
	})
	if err != nil {
		ctx.String(http.StatusOK, "更新失败!")
		zap.L().Error("用户信息更新失败:", zap.Error(err))
		return
	}
	ctx.String(http.StatusOK, "更新成功")
}

func (c *UserHandler) Profile(ctx *gin.Context) {
	type Profile struct {
		Email    string
		NickName string
		Birthday string
		AboutMe  string
	}

	id := ctx.MustGet("userid").(int64)
	u, err := c.svc.Profile(ctx, id)
	if err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	ctx.JSON(http.StatusOK, Profile{
		Email:    u.Email,
		NickName: u.NickName,
		Birthday: u.Birthday,
		AboutMe:  u.AboutMe,
	})
}
