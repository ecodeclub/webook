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

package interview

import (
	"sync"

	"github.com/ecodeclub/webook/internal/interview/internal/repository"
	"github.com/ecodeclub/webook/internal/interview/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/interview/internal/service"
	"github.com/ecodeclub/webook/internal/interview/internal/web"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"gorm.io/gorm"
)

type (
	JourneyHandler = web.InterviewJourneyHandler
	RoundHandler   = web.InterviewRoundHandler
)

func InitModule(db *egorm.Component) (*Module, error) {
	wire.Build(
		initJourneyDAO,
		initRoundDAO,
		repository.NewInterviewRoundRepository,
		repository.NewInterviewJourneyRepository,
		service.NewInterviewJourneyService,
		service.NewInterviewRoundService,
		web.NewInterviewJourneyHandler,
		web.NewInterviewRoundHandler,
		wire.Struct(new(Module), "*"),
	)
	return nil, nil
}

var initOnce sync.Once

func initJourneyDAO(db *gorm.DB) dao.InterviewJourneyDAO {
	initOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
	return dao.NewGORMInterviewJourneyDAO(db)
}

func initRoundDAO(db *gorm.DB) dao.InterviewRoundDAO {
	initOnce.Do(func() {
		err := dao.InitTables(db)
		if err != nil {
			panic(err)
		}
	})
	return dao.NewGORMInterviewRoundDAO(db)
}
