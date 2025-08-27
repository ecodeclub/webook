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

package comment

import (
	"sync"

	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/comment/internal/event"
	"github.com/ecodeclub/webook/internal/comment/internal/repository"
	"github.com/ecodeclub/webook/internal/comment/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/comment/internal/service"
	"github.com/ecodeclub/webook/internal/comment/internal/web"
	"github.com/ecodeclub/webook/internal/user"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
)

func InitModule(
	db *egorm.Component,
	q mq.MQ,
	userModule *user.Module) (*Module, error) {
	wire.Build(
		initCommentDAO,
		repository.NewCommentRepository,
		service.NewCommentService,
		event.NewQYWeChatEventProducer,
		web.NewHandler,
		wire.FieldsOf(new(*user.Module), "Svc"),
		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
}

var once = &sync.Once{}

func initCommentDAO(db *egorm.Component) (dao.CommentDAO, error) {
	var err error
	once.Do(func() {
		err = dao.InitTables(db)
	})
	if err != nil {
		return nil, err
	}
	return dao.NewCommentGORMDAO(db), nil
}
