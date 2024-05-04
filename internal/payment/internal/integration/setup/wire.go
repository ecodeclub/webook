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
	"github.com/ecodeclub/webook/internal/payment/internal/service/wechat"
	"github.com/ecodeclub/webook/internal/payment/ioc"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	testioc "github.com/ecodeclub/webook/internal/test/ioc"
	"github.com/google/wire"
	"gorm.io/gorm"
)

var serviceSet = wire.NewSet(
	testioc.BaseSet,
	initWechatConfig,
	ioc.InitWechatNativeService,
	InitDAO,
	repository.NewPaymentRepository,
	sequencenumber.NewGenerator,
	service.NewService,
)

func InitService(p event.PaymentEventProducer,
	cm *credit.Module,
	native wechat.NativeAPIService) payment.Service {
	wire.Build(
		serviceSet,
		wire.FieldsOf(new(*credit.Module), "Svc"),
	)
	return nil
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

func initWechatConfig() ioc.WechatConfig {
	return ioc.WechatConfig{
		AppID: "MockAPPID",
		MchID: "MockMchID",
	}
}
