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

package credit

import (
	"context"
	"sync"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit/internal/domain"
	"github.com/ecodeclub/webook/internal/credit/internal/event"
	"github.com/ecodeclub/webook/internal/credit/internal/repository"
	"github.com/ecodeclub/webook/internal/credit/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/credit/internal/service"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

type Credit = domain.Credit
type Service = service.Service

func InitModule(db *egorm.Component, q mq.MQ, e ecache.Cache) (*Module, error) {
	wire.Build(wire.Struct(
		new(Module), "*"),
		InitService,
		initCreditConsumer,
	)
	return new(Module), nil
}

var (
	once = &sync.Once{}
	svc  service.Service
)

func InitService(db *egorm.Component) Service {
	once.Do(func() {
		_ = dao.InitTables(db)
		d := dao.NewCreditGORMDAO(db)
		r := repository.NewCreditRepository(d)
		svc = service.NewCreditService(r)
	})
	return svc
}

func initCreditConsumer(svc service.Service, q mq.MQ) *event.CreditIncreaseConsumer {
	c, err := event.NewCreditIncreaseConsumer(svc, q)
	if err != nil {
		panic(err)
	}
	c.Start(context.Background())
	return c
}
