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
	"sync"
	"time"

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

var (
	once = &sync.Once{}
	svc  service.Service
)

func initService(db *egorm.Component) Service {
	once.Do(func() {
		_ = dao.InitTables(db)
		d := dao.NewMemberGORMDAO(db)
		r := repository.NewMemberRepository(d)
		svc = service.NewMemberService(r)
	})
	return svc
}

func InitService(db *egorm.Component) Service {
	wire.Build(initService)
	return nil
}

func InitMQConsumer(db *egorm.Component, c mq.Consumer) *event.MQConsumer {
	startAtFunc := func() int64 {
		return time.Now().Unix()
	}
	endAtFunc := func() int64 {
		return time.Date(2024, 6, 30, 23, 59, 59, 0, time.Local).Unix()
	}
	return event.NewMQConsumer(initService(db), c, startAtFunc, endAtFunc)
}
