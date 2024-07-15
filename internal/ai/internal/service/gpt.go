package service

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler"
)

//go:generate mockgen -source=./gpt.go -destination=../../mocks/gpt.mock.go -package=aimocks -typed=true GPTService
type GPTService interface {
	Invoke(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error)
}

type gptService struct {
	handlerFunc handler.HandleFunc
}

func NewGPTService(handlers []handler.GptHandler) GPTService {
	var hdl handler.HandleFunc
	for i := len(handlers) - 1; i >= 0; i-- {
		hdl = handlers[i].Next(hdl)
	}
	return &gptService{
		handlerFunc: hdl,
	}
}

func (g *gptService) Invoke(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
	return g.handlerFunc(ctx, req)
}
