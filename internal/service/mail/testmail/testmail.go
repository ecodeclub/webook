package testmail

import "context"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Send(ctx context.Context, from, subject, body, to string, isHTML bool) error {
	return nil
}
