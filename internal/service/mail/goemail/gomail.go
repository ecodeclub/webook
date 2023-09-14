package goemail

import (
	"context"

	"gopkg.in/gomail.v2"
)

type Service struct {
	cli gomail.SendCloser // 有接口用接口
}

func NewService(from string, cli gomail.SendCloser) *Service {
	return &Service{
		cli: cli,
	}
}

func (s *Service) Send(_ context.Context, from, subject, body, to string, _ bool) error {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	if err := gomail.Send(s.cli, m); err != nil {
		return err
	}
	return nil
}
