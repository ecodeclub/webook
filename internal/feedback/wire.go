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

package feedback

import (
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/feedback/internal/event"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/feedback/internal/service"
	"github.com/ecodeclub/webook/internal/feedback/internal/web"

	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitHandler(db *egorm.Component, q mq.MQ) (*Handler, error) {
	wire.Build(
		event.NewIncreaseCreditsEventProducer,
		InitService,
		web.NewHandler,
	)
	return new(Handler), nil
}

func InitService(db *egorm.Component, p event.IncreaseCreditsEventProducer) service.Service {
	wire.Build(
		initFeedbackDAO,
		repository.NewFeedBackRepository,
		service.NewFeedbackService,
	)
	return nil
}

var (
	daoOnce = sync.Once{}
	d       dao.FeedbackDAO
)

func initFeedbackDAO(db *gorm.DB) dao.FeedbackDAO {
	daoOnce.Do(func() {
		_ = dao.InitTables(db)
		d = dao.NewFeedbackDAO(db)
	})
	return d
}

type Handler = web.Handler
