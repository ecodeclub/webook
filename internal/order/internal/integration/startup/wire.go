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
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/order/internal/web"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/product"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
)

type Module struct {
	Handler      *web.Handler
	AdminHandler *web.AdminHandler
}

func InitModule(pm *payment.Module, ppm *product.Module, cm *credit.Module) (*Module, error) {
	wire.Build(testioc.BaseSet,
		order.InitService,
		order.InitHandler,
		web.NewAdminHandler,
		wire.Struct(new(Module), "*"),
	)

	return new(Module), nil
}
