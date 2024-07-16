package handler

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
)

type HandleFunc func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error)

func (f HandleFunc) Handle(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
	return f(ctx, req)
}

//go:generate mockgen -source=./type.go -destination=./mocks/handler.mock.go -package=hdlmocks -typed=true Handler
type Handler interface {
	Handle(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error)
}

type Builder interface {
	Next(next Handler) Handler
}
