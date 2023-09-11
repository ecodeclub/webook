package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/ecodeclub/webook/internal/domain"
	"github.com/ecodeclub/webook/internal/repository"
	"github.com/ecodeclub/webook/internal/repository/dao"
	"github.com/go-gomail/gomail"
	jwt "github.com/golang-jwt/jwt/v5"
	"os"
	"strconv"
	"time"
)

var (
	EmailVerified     = byte('1')
	EmailSubject      = "验证小微书帐户"
	EmailContent      = "我们需要先验证您的电子邮件地址，然后您才能开始使用小微书。此链接将在60分钟后到期。"
	EmailFoot         = "这是一封运营电子邮件。请勿回复此电子邮件。 我们将不会阅读或回应对此电子邮件的回复。"
	ErrEmailVertified = errors.New("请勿重复验证")
	ErrTokenInvalid   = errors.New("token不合法")
)

type EmailService interface {
	Send(ctx context.Context, email string) error
	Verify(ctx context.Context, token string) error
}

type UserEmailService struct {
	r repository.EamilRepository
}

func NewUserEmailService(r repository.EamilRepository) EmailService {
	return &UserEmailService{
		r: r,
	}
}

func (svc *UserEmailService) userToDomain(u dao.User) domain.User {
	return domain.User{
		Id:          u.Id,
		EmailVerify: u.EmailVerify.Byte,
		Email:       u.Email,
		Password:    u.Password,
	}
}

func (svc *UserEmailService) Verify(ctx context.Context, tokenStr string) error {
	ec := EmailClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, &ec, func(token *jwt.Token) (interface{}, error) {
		return JWTKey, nil
	})
	//此处已判断超时，也属于token非法的一种
	if err != nil || !token.Valid {
		return ErrTokenInvalid
	}

	user, err := svc.r.FindByEmail(ctx, ec.Email)
	if err != nil {
		return err
	}
	if user.EmailVerify.Valid {
		return ErrEmailVertified
	}
	user.EmailVerify.Byte = EmailVerified
	return svc.r.Update(ctx, svc.userToDomain(user))
}

func (svc *UserEmailService) Send(ctx context.Context, email string) error {
	//服务邮箱地址
	host := os.Getenv("EmailHost")
	//服务邮箱端口
	portstr := os.Getenv("EmailPort")
	//发送者或邮箱名称
	username := os.Getenv("EmailUsername")
	//邮箱的授权密码
	password := os.Getenv("EmailPassword")
	//小红书后台服务地址
	webookSever := os.Getenv("WebhookServer")

	port, _ := strconv.Atoi(portstr)

	var sendTo []string
	sendTo = append(sendTo, email)
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, EmailClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 60)),
		},
	})

	tokenStr, err := token.SignedString(JWTKey)
	if err != nil {
		return err
	}

	url := webookSever + "/users/emailverify/" + tokenStr
	body := fmt.Sprintf(`%s%s<a href="%s">%s</a>%s%s`,
		EmailContent, "<br><br>", url, "验证邮箱", "<br><br><br>", EmailFoot)

	m := gomail.NewMessage()
	m.SetHeader("From", username)
	m.SetHeader("To", sendTo...)

	m.SetHeader("Subject", EmailSubject)
	m.SetBody("text/html", body)

	d := gomail.NewDialer(host, port, username, password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err := d.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
