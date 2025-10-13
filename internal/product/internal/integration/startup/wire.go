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
	"github.com/ecodeclub/webook/internal/product"
	"github.com/ecodeclub/webook/internal/product/internal/service"
	"github.com/ecodeclub/webook/internal/product/internal/web"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

//func InitHandler() (*web.Handler, error) {
//	wire.Build(testioc.BaseSet, product.InitHandler)
//	return new(web.Handler), nil
//}

func InitService() service.Service {
	wire.Build(testioc.BaseSet, product.InitService)
	return nil
}

func InitHandler() (*web.Handler, error) {
	wire.Build(testioc.BaseSet, product.InitModule,
		wire.FieldsOf(new(*product.Module), "Hdl"))
	return new(web.Handler), nil
}
