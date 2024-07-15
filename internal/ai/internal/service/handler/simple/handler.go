package simple

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler"
)

// 最简业务handler
type Handler struct {
	handlerFunc handler.HandleFunc
}

func (h *Handler) Name() string {
	return "simple"
}

func (h *Handler) Next(next handler.HandleFunc) handler.HandleFunc {
	return func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		return h.handlerFunc(ctx, req)
	}
}
