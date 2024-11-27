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

package resume

import (
	"sync"

	"github.com/ecodeclub/webook/internal/cases"
	"github.com/ecodeclub/webook/internal/resume/internal/repository"
	"github.com/ecodeclub/webook/internal/resume/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/resume/internal/service"
	"github.com/ecodeclub/webook/internal/resume/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

func InitModule(db *egorm.Component, caModule *cases.Module) *Module {
	wire.Build(
		initResumeProjectDAOOnce,
		repository.NewResumeProjectRepo,
		service.NewService,
		wire.FieldsOf(new(*cases.Module), "ExamineSvc"),
		wire.FieldsOf(new(*cases.Module), "Svc"),
		web.NewHandler,
		wire.Struct(new(Module), "*"),
	)
	return new(Module)
}

var (
	resumeProjectDAO     dao.ResumeProjectDAO
	resumeProjectDAOOnce sync.Once
)

func initResumeProjectDAOOnce(db *egorm.Component) dao.ResumeProjectDAO {
	resumeProjectDAOOnce.Do(func() {
		resumeProjectDAO = dao.NewResumeProjectDAO(db)
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
	return resumeProjectDAO
}
