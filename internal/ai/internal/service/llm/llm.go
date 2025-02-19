package llm

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
)

//go:generate mockgen -source=./llm.go -destination=../../../mocks/llm.mock.go -package=aimocks -typed=true Service
type Service interface {
	Invoke(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error)
}

type llmService struct {
	// 这边显示依赖 FacadeHandler
	handler handler.Handler
}

func NewLLMService(root handler.Handler) Service {
	return &llmService{
		handler: root,
	}
}

func (g *llmService) Invoke(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error) {
	return g.handler.Handle(ctx, req)
}

