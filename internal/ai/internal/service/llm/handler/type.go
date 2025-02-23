package handler

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
)

type HandleFunc func(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error)

func (f HandleFunc) Handle(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error) {
	return f(ctx, req)
}

//go:generate mockgen -source=./type.go -destination=./mocks/handler.mock.go -package=hdlmocks -typed=true Handler
type Handler interface {
	Handle(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error)
}

type Builder interface {
	Next(next Handler) Handler
}

//go:generate mockgen -source=./type.go -destination=./stream_mocks/stream_handler.mock.go -package=hdlmocks -typed=true StreamHandler
type StreamHandler interface {
	StreamHandle(ctx context.Context, req domain.LLMRequest) (chan domain.StreamEvent, error)
}
