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

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler"
)

type QuestionExamineBizHandlerBuilder struct {
}

func NewQuestionExamineBizHandlerBuilder() *QuestionExamineBizHandlerBuilder {
	return &QuestionExamineBizHandlerBuilder{}
}

func (h *QuestionExamineBizHandlerBuilder) Next(next handler.Handler) handler.Handler {
	return handler.HandleFunc(func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		// 把 input 和 prompt 结合起来
		prompt := fmt.Sprintf(req.Config.PromptTemplate, req.Input[0], req.Input[1])
		req.Prompt = prompt
		return next.Handle(ctx, req)
	})
}
