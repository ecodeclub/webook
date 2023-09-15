package email

import (
	"context"
	"fmt"
)

type NoOpService struct {
}

func (n NoOpService) Send(ctx context.Context, subject, to string, content []byte) error {
	fmt.Println(string(content))
	return nil
}
