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

//go:build wireinject

package kbase

import (
	"net/http"
	"time"

	"github.com/ecodeclub/webook/internal/kbase/internal/domain"
	"github.com/ecodeclub/webook/internal/kbase/internal/service"
	"github.com/ecodeclub/webook/internal/kbase/internal/service/syncer"
	"github.com/ecodeclub/webook/internal/kbase/internal/web"
	"github.com/ecodeclub/webook/internal/roadmap"
	"github.com/gotomicro/ego/core/econf"

	baguwen "github.com/ecodeclub/webook/internal/question"

	"github.com/google/wire"
)

func InitModule(queModule *baguwen.Module, rdModule *roadmap.Module) *Module {
	wire.Build(
		initConfig,
		initService,
		wire.FieldsOf(new(*baguwen.Module), "Svc"),
		wire.FieldsOf(new(*roadmap.Module), "AdminSvc"),
		initSyncerMap,
		service.NewSyncService,
		web.NewAdminHandler,
		wire.Struct(new(Module), "*"),
	)
	return new(Module)
}

func initConfig() Cfg {
	var cfg Cfg
	err := econf.UnmarshalKey("kbase", &cfg)
	if err != nil {
		panic(err)
	}
	return cfg
}

func initService(cfg Cfg) service.Service {
	return service.NewHTTPKBaseService(cfg.BaseURL, &http.Client{}, cfg.BatchSize,
		cfg.RetryStrategy.Interval, cfg.RetryStrategy.MaxInterval, cfg.RetryStrategy.MaxRetries)
}

func initSyncerMap(cfg Cfg, baguwenSvc baguwen.Service, rdSvc roadmap.AdminService, svc service.Service) map[string]service.Syncer {
	return map[string]service.Syncer{
		domain.BizQuestion: syncer.NewQuestionSyncer(cfg.QuestionSyncer.IndexName,
			cfg.BatchSize, baguwenSvc, svc),
		domain.BizQuestionRel: syncer.NewQuestionRelSyncer(cfg.QuestionRelSyncer.IndexName,
			cfg.BatchSize, rdSvc, svc),
	}
}

type SyncerConfig struct {
	IndexName string `json:"indexName"`
}
type RetryStrategy struct {
	Interval    time.Duration `json:"interval"`
	MaxInterval time.Duration `json:"maxInterval"`
	MaxRetries  int           `json:"maxRetries"`
}
type Cfg struct {
	BaseURL           string        `json:"baseURL"`
	BatchSize         int           `json:"batchSize"`
	QuestionSyncer    SyncerConfig  `json:"questionSyncer"`
	QuestionRelSyncer SyncerConfig  `json:"questionRelSyncer"`
	RetryStrategy     RetryStrategy `json:"retryStrategy"`
}
