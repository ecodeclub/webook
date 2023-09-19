package gomail

import (
	"context"

	"github.com/ecodeclub/webook/internal/service/email"

	"github.com/go-gomail/gomail"
)

type EmailServic struct {
	d *gomail.Dialer
}

func NewEmailService(dialer *gomail.Dialer) email.Service {
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
