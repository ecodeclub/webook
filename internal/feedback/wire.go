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

	"github.com/ecodeclub/webook/internal/feedback/internal/repository"
	"github.com/ecodeclub/webook/internal/feedback/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/feedback/internal/service"
	"github.com/ecodeclub/webook/internal/feedback/internal/web"

	"github.com/ecodeclub/ecache"

	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

func InitHandler(db *egorm.Component, ec ecache.Cache) (*Handler, error) {
	wire.Build(
		InitFeedBackDAO,
		repository.NewFeedBackRepo,
		service.NewService,
		web.NewHandler,
	)
	return new(Handler), nil
}

var daoOnce = sync.Once{}

func InitTableOnce(db *gorm.DB) {
	daoOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
}

func InitFeedBackDAO(db *egorm.Component) dao.FeedbackDAO {
	InitTableOnce(db)
	return dao.NewFeedBackDAO(db)
}

type Handler = web.Handler
