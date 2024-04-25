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
	"sync"

	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/payment/internal/event"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
	credit2 "github.com/ecodeclub/webook/internal/payment/internal/service/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/service/wechat"
	"github.com/ecodeclub/webook/internal/payment/internal/web"
	"github.com/ecodeclub/webook/internal/payment/ioc"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
	"github.com/gotomicro/ego/core/elog"
	"gorm.io/gorm"
)

func InitModule(p event.PaymentEventProducer,
	paymentDDLFunc func() int64,
	cm *credit.Module,
	h wechat.NotifyHandler,
	native wechat.NativeAPIService) *payment.Module {
	wire.Build(
		testioc.BaseSet,
		initLogger,
		initWechatConfig,
		ioc.InitWechatNativeService,
		InitDAO,
		web.NewHandler,
		service.NewService,
		credit2.NewCreditPaymentService,
		repository.NewPaymentRepository,
		sequencenumber.NewGenerator,
		wire.FieldsOf(new(*credit.Module), "Svc"),
		wire.Struct(new(payment.Module), "*"),
	)
	return new(payment.Module)
}

var (
	once       = &sync.Once{}
	paymentDAO dao.PaymentDAO
)

func InitDAO(db *gorm.DB) dao.PaymentDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
		paymentDAO = dao.NewPaymentGORMDAO(db)
	})
	return paymentDAO
}

func initLogger() *elog.Component {
	return elog.DefaultLogger
}

func initWechatConfig() ioc.WechatConfig {
	return ioc.WechatConfig{
		AppID: "MockAPPID",
		MchID: "MockMchID",
	}
}
