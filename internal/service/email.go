package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/service/mail"
)

type EmailService interface {
	Send(ctx context.Context, from, subject, body, to string, isHTML bool) error
}

type emailService struct {
	mailSvc mail.Service
}

func NewEmailService(mailSvc mail.Service) EmailService {
	return &emailService{
		mailSvc: mailSvc,
	}
}

// Send 发送邮件。考虑到不同的请求所需要发送的内容可能都不相同，因此不在这里过多处理。
func (e *emailService) Send(ctx context.Context,
	from, subject, body, to string, isHTML bool) error {
	return e.mailSvc.Send(ctx, from, subject, body, to, isHTML)
}
