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

package ai

import (
	"github.com/ecodeclub/webook/internal/ai/internal/repository"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/config"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/credit"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/log"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/platform/ali_deepseek"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/platform/zhipu"
	"github.com/ecodeclub/webook/internal/ai/internal/service/llm/handler/record"
	"github.com/gotomicro/ego/core/econf"
)

func InitCompositionHandlerUsingZhipu(common []handler.Builder,
	root *zhipu.Handler) handler.Handler {
	return handler.NewCompositionHandler(common, root)
}

func InitZhipu() *zhipu.Handler {
	type Config struct {
		APIKey string `yaml:"apikey"`
	}
	var cfg Config
	err := econf.UnmarshalKey("zhipu", &cfg)
	if err != nil {
		panic(err)
	}
	h, err := zhipu.NewHandler(cfg.APIKey)
	if err != nil {
		panic(err)
	}
	return h
}

func InitCommonHandlers(log *log.HandlerBuilder,
	cfg *config.HandlerBuilder,
	credit *credit.HandlerBuilder,
	record *record.HandlerBuilder) []handler.Builder {
	return []handler.Builder{log, cfg, credit, record}
}

func InitAliDeepSeekHandler(configRepo repository.ConfigRepository, logRepo repository.LLMLogRepo) handler.StreamHandler {
	type Config struct {
		APIKey string `yaml:"apikey"`
	}
	var cfg Config
	err := econf.UnmarshalKey("ali", &cfg)
	if err != nil {
		panic(err)
	}
	return ali_deepseek.NewHandler(cfg.APIKey, logRepo, configRepo)
}
