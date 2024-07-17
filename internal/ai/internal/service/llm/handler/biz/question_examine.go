// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package biz

import (
	"context"
	"fmt"
	"unicode/utf8"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler"
)

type QuestionExamineBizHandlerBuilder struct {
}

func NewQuestionExamineBizHandlerBuilder() *QuestionExamineBizHandlerBuilder {
	return &QuestionExamineBizHandlerBuilder{}
}

func (h *QuestionExamineBizHandlerBuilder) Next(next handler.Handler) handler.Handler {
	return handler.HandleFunc(func(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error) {
		title := req.Input[0]
		userInput := req.Input[1]
		userInputLen := utf8.RuneCount([]byte(userInput))

		if userInputLen > req.Config.MaxInput {
			return domain.LLMResponse{}, fmt.Errorf("输入太长，最常不超过 %d，现有长度 %d", req.Config.MaxInput, userInputLen)
		}
		// 把 input 和 prompt 结合起来
		prompt := fmt.Sprintf(req.Config.PromptTemplate, title, userInput)
		req.Prompt = prompt
		return next.Handle(ctx, req)
	})
}
