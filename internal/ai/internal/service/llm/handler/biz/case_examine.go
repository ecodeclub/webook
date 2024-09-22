package biz

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler"
)

type CaseExamineBizHandlerBuilder struct {
}

func NewCaseExamineBizHandlerBuilder() *CaseExamineBizHandlerBuilder {
	return &CaseExamineBizHandlerBuilder{}
}

func (h *CaseExamineBizHandlerBuilder) Next(next handler.Handler) handler.Handler {
	return handler.HandleFunc(func(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error) {
		title := req.Input[0]
		refCase := req.Input[1]
		userInput := req.Input[2]
		userInputLen := utf8.RuneCount([]byte(userInput))

		if userInputLen > req.Config.MaxInput {
			return domain.LLMResponse{}, fmt.Errorf("输入太长，最常不超过 %d，现有长度 %d", req.Config.MaxInput, userInputLen)
		}
		// 把 input 和 prompt 结合起来
		prompt := fmt.Sprintf(req.Config.PromptTemplate, title, refCase, userInput)
		req.Prompt = prompt
		return next.Handle(ctx, req)
	})
}
