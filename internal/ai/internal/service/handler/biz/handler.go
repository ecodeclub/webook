package biz

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler"
)

// 用于分发业务Biz
type FacadeHandler struct {
	bizMap map[string]handler.GptHandler
}

func (h *FacadeHandler) Name() string {
	return "biz_facade"
}

func NewHandler(bizMap map[string]handler.GptHandler) *FacadeHandler {
	return &FacadeHandler{
		bizMap: bizMap,
	}
}

func (h *FacadeHandler) Next(next handler.HandleFunc) handler.HandleFunc {
	return func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		handleFunc, ok := h.bizMap[req.Biz]
		if !ok {
			return domain.GPTResponse{}, handler.ErrUnknownBiz
		}
		nextFunc := handleFunc.Next(next)
		return nextFunc(ctx, req)
	}
}
