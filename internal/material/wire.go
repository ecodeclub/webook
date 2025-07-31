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

package material

import (
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/material/internal/event"
	"github.com/ecodeclub/webook/internal/material/internal/repository"
	"github.com/ecodeclub/webook/internal/material/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/material/internal/service"
	"github.com/ecodeclub/webook/internal/material/internal/web"
	"github.com/ecodeclub/webook/internal/sms/client"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

type (
	Handler      = web.Handler
	AdminHandler = web.AdminHandler
)

func InitModule(db *egorm.Component, q mq.MQ, client client.Client, userModule *user.Module) (*Module, error) {
	wire.Build(
		initDAO,
		repository.NewMaterialRepository,
		event.NewMemberEventProducer,
		service.NewMaterialService,
		web.NewHandler,
		web.NewAdminHandler,
		wire.FieldsOf(new(*user.Module), "Svc"),
		wire.Struct(new(Module), "*"),
	)
	return nil, nil
}

var initOnce sync.Once

func initDAO(db *gorm.DB) dao.MaterialDAO {
	initOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
	return dao.NewGORMMaterialDAO(db)
}
