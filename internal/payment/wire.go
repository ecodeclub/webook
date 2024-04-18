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

package payment

import (
	"sync"
	"time"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
	"github.com/ecodeclub/webook/internal/payment/internal/event"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
	credit2 "github.com/ecodeclub/webook/internal/payment/internal/service/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/web"
	"github.com/ecodeclub/webook/internal/payment/ioc"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"github.com/gotomicro/ego/core/elog"
	"gorm.io/gorm"
)

type Handler = web.Handler
type Payment = domain.Payment
type Record = domain.PaymentRecord
type Channel = domain.PaymentChannel
type ChannelType = domain.ChannelType

var ChannelTypeCredit = domain.ChannelTypeCredit
var ChannelTypeWechat = domain.ChannelTypeWechat

type Service = service.Service

func InitModule(db *egorm.Component,
	mq mq.MQ,
	c ecache.Cache,
	cm *credit.Module) (*Module, error) {
	wire.Build(
		initLogger,
		ioc.InitWechatNativeService,
		ioc.InitWechatConfig,
		ioc.InitWechatNotifyHandler,
		ioc.InitWechatClient,
		initDAO,
		initPaymentEventProducer,
		web.NewHandler,
		service.NewService,
		credit2.NewCreditPaymentService,
		repository.NewPaymentRepository,
		paymentDDLFunc,
		sequencenumber.NewGenerator,
		wire.FieldsOf(new(*credit.Module), "Svc"),
		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
}

var (
	once     = &sync.Once{}
	orderDAO dao.PaymentDAO
)

func initPaymentEventProducer(mq mq.MQ) (event.PaymentEventProducer, error) {
	p, err := mq.Producer("payment_events")
	if err != nil {
		return nil, err
	}
	return event.NewPaymentEventProducer(p)
}

func paymentDDLFunc() func() int64 {
	return func() int64 {
		return time.Now().Add(time.Minute * 30).UnixMilli()
	}
}

func initDAO(db *gorm.DB) dao.PaymentDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
		orderDAO = dao.NewPaymentGORMDAO(db)
	})
	return orderDAO
}

func initLogger() *elog.Component {
	return elog.DefaultLogger
}
