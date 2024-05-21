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

package marketing

import (
	"context"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/consumer"
	"github.com/ecodeclub/webook/internal/marketing/internal/event/producer"
	"github.com/ecodeclub/webook/internal/product"
	"github.com/lithammer/shortuuid/v4"

	"github.com/ecodeclub/webook/internal/marketing/internal/repository"
	"github.com/ecodeclub/webook/internal/marketing/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/marketing/internal/service"
	"github.com/ecodeclub/webook/internal/marketing/internal/web"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

type (
	Service      = service.Service
	Handler      = web.Handler
	AdminHandler = web.AdminHandler
)

func InitModule(db *egorm.Component, q mq.MQ, om *order.Module, pm *product.Module) (*Module, error) {
	wire.Build(
		dao.NewGORMMarketingDAO,
		repository.NewRepository,
		wire.FieldsOf(new(*order.Module), "Svc"),
		wire.FieldsOf(new(*product.Module), "Svc"),
		sequencenumber.NewGenerator,
		redemptionCodeGenerator,
		eventKeyGenerator,
		producer.NewMemberEventProducer,
		producer.NewCreditEventProducer,
		producer.NewPermissionEventProducer,
		service.NewService,
		web.NewHandler,
		service.NewAdminService,
		web.NewAdminHandler,
		newOrderEventConsumer,
		wire.Struct(new(Module), "*"),
	)
	return nil, nil
}

func newOrderEventConsumer(svc service.Service, q mq.MQ) (*consumer.OrderEventConsumer, error) {
	res, err := consumer.NewOrderEventConsumer(svc, q)
	if err == nil {
		res.Start(context.Background())
	}
	return res, err
}

func redemptionCodeGenerator(generator *sequencenumber.Generator) func(id int64) string {
	return func(id int64) string {
		code, _ := generator.Generate(id)
		return code
	}
}

func eventKeyGenerator() func() string {
	return shortuuid.New
}
