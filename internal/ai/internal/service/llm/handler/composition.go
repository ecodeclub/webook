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

package handler

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
)

// CompositionHandler 通过组合 Handler 来完成某个业务
// 后续该部分应该是动态计算的，通过结合配置来实现动态计算
type CompositionHandler struct {
	root Handler
}

func (c *CompositionHandler) Handle(ctx context.Context, req domain.LLMRequest) (domain.LLMResponse, error) {
	return c.root.Handle(ctx, req)
}

func NewCompositionHandler(common []Builder,
	root Handler) *CompositionHandler {
	for i := len(common) - 1; i >= 0; i-- {
		current := common[i]
		root = current.Next(root)
	}
	return &CompositionHandler{
		root: root,
	}
}
