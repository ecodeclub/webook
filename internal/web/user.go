package web

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	regexp "github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/service"
	tokenGen "github.com/ecodeclub/webook/internal/web/token/generator"
	tokenVfy "github.com/ecodeclub/webook/internal/web/token/validator"
)

var (
	CodeParamsErr         = 401000 // 参数错误
	CodeEmailVerifyFailed = 401001 // 邮箱验证失败 4:参数错误 01:用户服务
	CodeEmailVerified     = 401002 // 邮箱已验证
)

const (
	// 密码规则：长度至少 6 位
	passwordRegexPattern = `^.{6,}$`
)

type UserHandler struct {
	svc               service.UserService
	emailSvc          service.EmailService
	passwordRegexExp  *regexp.Regexp
	emailVerifyGen    tokenGen.TokenGenerator
	emailVerifier     tokenVfy.Verifier
	emailVerifyABSURL string // 前端的绝对URL
	logger            *zap.Logger
}

func NewUserHandler(svc service.UserService, emailSvc service.EmailService,
	emailVerifyGen tokenGen.TokenGenerator, emailVerifier tokenVfy.Verifier,
	emailVerifyURL string, logger *zap.Logger) *UserHandler {
	return &UserHandler{
		svc:               svc,
		emailSvc:          emailSvc,
		passwordRegexExp:  regexp.MustCompile(passwordRegexPattern, regexp.None),
		emailVerifyGen:    emailVerifyGen,
		emailVerifier:     emailVerifier,
		emailVerifyABSURL: emailVerifyURL,
		logger:            logger,
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

	// 密码和确认密码
	if info.Password != info.ConfirmPassword {
		ctx.String(http.StatusBadRequest, "两次密码不相同！")
		return
	}
	// 密码规律
	pwdFlag, err := u.passwordRegexExp.MatchString(info.Password)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统错误！")
		return
	}
	if !pwdFlag {
		ctx.String(http.StatusBadRequest, "密码格式不正确,长度不能小于 6 位！")
		return
	}

	// 存储数据...
	err = u.svc.Signup(ctx.Request.Context(), &domain.User{
		Email:    info.Email,
		Password: info.Password,
	})
	if errors.Is(err, service.ErrUserDuplicate) {
		ctx.String(http.StatusBadRequest, "重复邮箱，请更换邮箱！")
		return
	}
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统错误！")
		return
	}

	go func() {
		// 生成一个10分钟的token
		token, err := u.emailVerifyGen.GenerateToken(info.Email, time.Duration(10)*time.Minute)
		if err != nil {
			u.logger.Error("生成token失败", zap.Error(err))
			return
		}

		// 组装短信内容
		fullURL, err := u.URLGenerator(u.emailVerifyABSURL, map[string][]string{"code": {token}})
		if err != nil {
			u.logger.Error("生成URL失败", zap.Error(err))
			return
		}
		emailBody := strings.Join([]string{
			"请点击下方链接验证邮箱。\n", fullURL, "\n10分钟内有效。",
		}, "")

		err = u.emailSvc.Send(context.Background(), "webook邮箱验证", "验证邮箱", emailBody, info.Email, false)
		if err != nil {
			u.logger.Error("发送邮件失败", zap.Error(err))
			return
		}
	}()
	ctx.String(http.StatusOK, "注册成功！")
}

func (u *UserHandler) EmailVerify(ctx *gin.Context) {
	type emailVerifyVO struct {
		Code string `form:"code" binding:"required,max=512"`
	}
	var vo emailVerifyVO
	err := ctx.ShouldBind(&vo)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: CodeParamsErr,
			Msg:  "参数错误",
		})
		return
	}

	email, err := u.emailVerifier.Verify(vo.Code)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, Result{
			Code: CodeEmailVerifyFailed,
			Msg:  "验证失败",
		})
		return
	}

	err = u.svc.EmailVerify(ctx.Request.Context(), email)
	if err != nil {
		if errors.Is(err, service.ErrUserEmailVerified) {
			ctx.JSON(http.StatusBadRequest, Result{
				Code: CodeEmailVerified,
				Msg:  "邮箱已验证",
			})
			return
		}
		ctx.JSON(http.StatusBadRequest, Result{
			Code: CodeEmailVerifyFailed,
			Msg:  "验证失败",
		})
		return
	}
	ctx.JSON(http.StatusOK, Result{Msg: "验证成功"})
}

// URLGenerator 生成一个URL。params可以foo=1&foo=2
// 示例：URLGenerator("http://example.com/api", map[string][]string{"foo": []string{"bar"}})
func (u *UserHandler) URLGenerator(absoluteURL string, params map[string][]string) (string, error) {
	uv := url.Values{}
	for k, value := range params {
		if len(value) <= 0 {
			continue
		}
		for _, v := range value {
			uv.Add(k, v)
		}
	}

	up, _ := url.Parse(absoluteURL) // 这个很难出现err
	if up == nil || !up.IsAbs() {
		return "", errors.New("绝对URL错误")
	}

	up.RawQuery = uv.Encode()
	return up.String(), nil
}
