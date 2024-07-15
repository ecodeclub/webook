package gpt

import (
	"context"
	"time"

	"github.com/ecodeclub/ekit/retry"
	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler/gpt/getter"
)

type Handler struct {
	sdkGetter getter.AiSdkGetter
	retryFunc *retry.ExponentialBackoffRetryStrategy
}

const (
	defaultMinRetryInterval = 100 * time.Millisecond
	defaultMaxRetryInterval = 10 * time.Second
	defaultMaxRetryTimes    = 10
)

func NewHandler(sdkGetter getter.AiSdkGetter) (*Handler, error) {
	strategy, err := retry.NewExponentialBackoffRetryStrategy(defaultMinRetryInterval, defaultMaxRetryInterval, defaultMaxRetryTimes)
	if err != nil {
		return nil, err
	}
	return &Handler{
		sdkGetter: sdkGetter,
		retryFunc: strategy,
	}, nil
}

func (h *Handler) Name() string {
	return "gpt"
}

func (h *Handler) Next(next handler.HandleFunc) handler.HandleFunc {
	return func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		for {
			gptSdk, err := h.sdkGetter.GetSdk(req.Biz)
			if err != nil {
				sleepTime, ok := h.retryFunc.Next()
				if ok {
					time.Sleep(sleepTime)
					continue
				} else {
					return domain.GPTResponse{}, err
				}
			}
			tokens, ans, err := gptSdk.Invoke(ctx, req.Input)
			if err != nil {
				sleepTime, ok := h.retryFunc.Next()
				if ok {
					time.Sleep(sleepTime)
					continue
				} else {
					return domain.GPTResponse{}, err
				}
			}
			return domain.GPTResponse{
				Tokens: int(tokens),
				Answer: ans,
			}, nil
		}
	}
}
