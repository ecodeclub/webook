package log

import (
	"context"
	"fmt"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/handler"
	"github.com/gotomicro/ego/core/elog"
)

type Handler struct {
	logger *elog.Component
}

func NewHandler() *Handler {
	return &Handler{
		logger: elog.DefaultLogger,
	}
}

func (h *Handler) Name() string {
	return "log"
}

func (h *Handler) Next(next handler.HandleFunc) handler.HandleFunc {
	return func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		// 记录请求
		h.logger.Info(fmt.Sprintf("请求gpt服务请求id为 %s", req.Tid), elog.FieldExtMessage(req))
		resp, err := next(ctx, req)
		if err != nil {
			// 记录错误
			h.logger.Error(fmt.Sprintf("请求gpt服务失败请求id为 %s", req.Tid), elog.FieldErr(err))
			return resp, err
		}
		// 记录响应
		h.logger.Info(fmt.Sprintf("请求gpt服务请求id为 %s", req.Tid), elog.FieldExtMessage(resp))
		return resp, err
	}
}
