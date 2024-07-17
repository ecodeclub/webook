package llm

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/biz"
)

//go:generate mockgen -source=./llm.go -destination=../../../mocks/llm.mock.go -package=aimocks -typed=true Service
type Service interface {
	Invoke(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error)
}

type llmService struct {
	// 这边显示依赖 FacadeHandler
	handler *biz.FacadeHandler
}

func NewLLMService(facade *biz.FacadeHandler) Service {
	return &llmService{
		handler: facade,
	}
}

func (g *llmService) Invoke(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error) {
	return g.handler.Handle(ctx, req)
}
