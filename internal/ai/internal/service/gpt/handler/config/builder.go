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

package config

import (
	"context"

	"github.com/ecodeclub/webook/internal/ai/internal/domain"
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/service/gpt/handler"
)

// HandlerBuilder 改为从数据库中读取
type HandlerBuilder struct {
	repo repository.ConfigRepository
}

func NewBuilder(repo repository.ConfigRepository) *HandlerBuilder {
	return &HandlerBuilder{
		repo: repo,
	}
}

func (b *HandlerBuilder) Next(next handler.Handler) handler.Handler {
	return handler.HandleFunc(func(ctx context.Context, req domain.GPTRequest) (domain.GPTResponse, error) {
		// 读取配置
		cfg, err := b.repo.GetConfig(ctx, req.Biz)
		if err != nil {
			return domain.GPTResponse{}, err
		}
		req.Config = cfg
		return next.Handle(ctx, req)
	})
}

var _ handler.Builder = &HandlerBuilder{}
