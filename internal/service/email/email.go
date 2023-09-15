package email

import (
	"context"
	"github.com/go-gomail/gomail"
)

type Service interface {
	Send(ctx context.Context, subject, to string, content []byte) error
}

type EmailServic struct {
	d *gomail.Dialer
}

func NewEmailService(dialer *gomail.Dialer) Service {
	return &EmailServic{
		d: dialer,
	}
}

func (svc *EmailServic) Send(ctx context.Context, subject, to string, content []byte) error {
	var sendTo []string
	sendTo = append(sendTo, to)

	m := gomail.NewMessage()
	m.SetHeader("From", svc.d.Username)
	m.SetHeader("To", sendTo...)

	m.SetHeader("Subject", subject)
	m.SetBody("text/html", string(content))

	return svc.d.DialAndSend(m)
}
