package email

import "context"

type Service interface {
	Send(ctx context.Context, subject, to string, content []byte) error
}
