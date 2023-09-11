package web

import (
	"net/http"

	regexp "github.com/dlclark/regexp2"
	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	//密码规则：长度至少 6 位
	passwordRegexPattern = `^.{6,}$`
)

type UserHandler struct {
	svc              service.UserAndService
	evc              service.EmailService
	passwordRegexExp *regexp.Regexp
}

func NewUserHandler(svc service.UserAndService, evc service.EmailService) *UserHandler {
	return &UserHandler{
		svc:              svc,
		evc:              evc,
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

	//发送验证邮箱的邮件
	err = u.evc.Send(ctx.Request.Context(), info.Email)
	if err != nil {
		// 应该有日志
		return
	}
}

func (u *UserHandler) EmailVerify(ctx *gin.Context) {
	token := ctx.Param("token")

	err := u.evc.Verify(ctx.Request.Context(), token)
	if err != nil {
		ctx.String(http.StatusOK, "验证失败!")
		return
	}
	ctx.String(http.StatusOK, "验证成功!")
}
