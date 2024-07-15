package config

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler"
)

type Handler struct {
	configMap map[string]domain.GPTBiz
}

func (h *Handler) Name() string {
	return "config"
}

func InitHandler() *Handler {
	cfgs := []domain.GPTBiz{
		{
			Biz:              "simple",
			AmountPerToken:   1,
			CreditPerToken:   1,
			MaxTokensPerTime: 1000,
		},
	}
	cfgMap := make(map[string]domain.GPTBiz, len(cfgs))
	for _, bizConfig := range cfgs {
		cfgMap[bizConfig.Biz] = bizConfig
	}
	return NewHandler(cfgMap)
}

func NewHandler(configMap map[string]domain.GPTBiz) *Handler {
	return &Handler{
		configMap: configMap,
	}
}

func (h *Handler) Next(next handler.HandleFunc) handler.HandleFunc {
	return func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		cfg, ok := h.configMap[req.Biz]
		if !ok {
			return domain.GPTResponse{}, handler.ErrUnknownBiz
		}
		req.BizConfig = cfg
		return next(ctx, req)
	}
}
