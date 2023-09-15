package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository"
	"github.com/ecodeclub/webook/internal/service/email"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrUserDuplicate = repository.ErrUserDuplicate
	emailSubject     = "验证小微书帐户"
	emailContent     = "我们需要先验证您的电子邮件地址，然后您才能开始使用小微书。此链接将在60分钟后到期。"
	emailFoot        = "这是一封运营电子邮件。请勿回复此电子邮件。 我们将不会阅读或回应对此电子邮件的回复。"
	webookSever      = "http://localhost:8080"
	ErrTokenInvalid  = errors.New("token不合法")
)

type UserService interface {
	Signup(ctx context.Context, u *domain.User) error
	SendVerifyEmail(ctx context.Context, email string) error
	VerifyEmail(ctx context.Context, tokenStr string) error
}

type userService struct {
	r        repository.UserRepository
	emailSvc email.Service
}

func NewUserService(r repository.UserRepository, e email.Service) UserService {
	return &userService{
		r:        r,
		emailSvc: e,
	}
}

func (svc *userService) Signup(ctx context.Context, u *domain.User) error {
	//hashPwd, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	//if err != nil {
	//	return err
	//}
	//u.Password = string(hashPwd)
	return svc.r.Create(ctx, u)
}

func (svc *userService) SendVerifyEmail(ctx context.Context, emailAddr string) error {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, EmailClaims{
		Email: emailAddr,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 60)),
		},
	})
	tokenStr, err := token.SignedString(EmailJWTKey)
	if err != nil {
		return err
	}

	url := webookSever + "/users/email/verify/" + tokenStr
	body := fmt.Sprintf(`%s%s<a href="%s">%s</a>%s%s`,
		emailContent, "<br><br>", url, "验证邮箱", "<br><br><br>", emailFoot)

	return svc.emailSvc.Send(ctx, emailSubject, emailAddr, []byte(body))
}

func (svc *userService) VerifyEmail(ctx context.Context, tokenStr string) error {
	ec := EmailClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, &ec, func(token *jwt.Token) (interface{}, error) {
		return EmailJWTKey, nil
	})
	//此处已判断超时，也属于token非法的一种
	if err != nil || !token.Valid {
		return ErrTokenInvalid
	}

	return svc.r.UpdateEmailVerified(ctx, ec.Email)
}
