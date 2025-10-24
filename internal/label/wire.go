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

package label

import (
	"sync"

	"github.com/ecodeclub/webook/internal/label/internal/repository"
	"github.com/ecodeclub/webook/internal/label/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/label/internal/service"
	"github.com/ecodeclub/webook/internal/label/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

type Module struct {
	AdminHandler *AdminHandler
	Handler      *Handler
}

var ModuleSet = wire.NewSet(
	repository.NewCachedLabelRepository,
	InitTablesOnce,
	service.NewService,
	web.NewHandler,
	web.NewAdminHandler,
)

func InitModule(db *egorm.Component) *Module {
	wire.Build(
		ModuleSet,
		wire.Struct(new(Module), "*"),
	)
	return new(Module)
}

var once = &sync.Once{}

func InitTablesOnce(db *egorm.Component) dao.LabelDAO {
	once.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
	return dao.NewLabelGORMDAO(db)
}

type AdminHandler = web.AdminHandler
type Handler = web.Handler
