package handler

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
)

type HandleFunc func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error)

type GptHandler interface {
	Name() string
	Next(next HandleFunc) HandleFunc
}
