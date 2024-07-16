package gpt

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler/biz"
)

//go:generate mockgen -source=./gpt.go -destination=../../../mocks/gpt.mock.go -package=aimocks -typed=true Service
type Service interface {
	Invoke(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error)
}

type gptService struct {
	// 这边显示依赖 FacadeHandler
	handler *biz.FacadeHandler
}

func NewGPTService(facade *biz.FacadeHandler) Service {
	return &gptService{
		handler: facade,
	}
}

func (g *gptService) Invoke(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
	return g.handler.Handle(ctx, req)
}
