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

package permission

import (
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/permission/internal/domain"
	"github.com/ecodeclub/webook/internal/permission/internal/event"
	"github.com/ecodeclub/webook/internal/permission/internal/repository"
	"github.com/ecodeclub/webook/internal/permission/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/permission/internal/service"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

type (
	Service            = service.Service
	PersonalPermission = domain.PersonalPermission
)

func InitModule(db *egorm.Component, q mq.MQ) (*Module, error) {
	wire.Build(
		initDAO,
		repository.NewPermissionRepository,
		service.NewPermissionService,
		event.NewPermissionEventConsumer,
		wire.Struct(new(Module), "*"),
	)
	return nil, nil
}

var (
	once          = &sync.Once{}
	permissionDAO dao.PermissionDAO
)

func initDAO(db *gorm.DB) dao.PermissionDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
		permissionDAO = dao.NewPermissionGORMDAO(db)
	})
	return permissionDAO
}
