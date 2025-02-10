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

package startup

import (
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/internal/interactive"
	"github.com/ecodeclub/webook/internal/permission"
	"github.com/ecodeclub/webook/internal/project"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

func InitModule(intrModule *interactive.Module, permModule *permission.Module, sp session.Provider) (*project.Module, error) {
	wire.Build(project.InitModule, testioc.InitDB, testioc.InitMQ)
	return new(project.Module), nil
}
