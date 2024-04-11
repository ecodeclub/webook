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

package member

import (
	"context"
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/member/internal/domain"
	"github.com/ecodeclub/webook/internal/member/internal/event"
	"github.com/ecodeclub/webook/internal/member/internal/repository"
	"github.com/ecodeclub/webook/internal/member/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/member/internal/service"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

type Member = domain.Member
type Service = service.Service

func InitModule(db *egorm.Component, q mq.MQ) (*Module, error) {
	wire.Build(wire.Struct(
		new(Module), "*"),
		InitService,
		initRegistrationConsumer,
	)
	return new(Module), nil
}

var (
	once = &sync.Once{}
	svc  service.Service
)

func InitService(db *egorm.Component, q mq.MQ) Service {
	once.Do(func() {
		_ = dao.InitTables(db)
		d := dao.NewMemberGORMDAO(db)
		r := repository.NewMemberRepository(d)
		svc = service.NewMemberService(r)
	})
	return svc
}

func initRegistrationConsumer(svc service.Service, q mq.MQ) *event.RegistrationEventConsumer {
	c, err := event.NewRegistrationEventConsumer(svc, q)
	if err != nil {
		panic(err)
	}
	c.Start(context.Background())
	return c
}
