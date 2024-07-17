package biz

import (
	"context"
	"errors"
	"fmt"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	handler2 "github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler"
)

var ErrUnknownBiz = errors.New("未知的业务")

// FacadeHandler 用于分发业务Biz
type FacadeHandler struct {
	bizMap map[string]handler2.Handler
}

func (f *FacadeHandler) Handle(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error) {
	h, ok := f.bizMap[req.Biz]
	if !ok {
		return domain.LLMResponse{}, fmt.Errorf("%w biz: %s", ErrUnknownBiz, req.Biz)
	}
	return h.Handle(ctx, req)
}

var _ handler2.Handler = &FacadeHandler{}

func NewHandler(bizMap map[string]handler2.Handler) *FacadeHandler {
	return &FacadeHandler{
		bizMap: bizMap,
	}
}
