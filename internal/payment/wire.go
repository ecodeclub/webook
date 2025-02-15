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

	"github.com/ecodeclub/webook/internal/user"

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/event"
	"github.com/ecodeclub/webook/internal/payment/internal/job"
	"github.com/ecodeclub/webook/internal/payment/internal/repository"
	"github.com/ecodeclub/webook/internal/payment/internal/repository/dao"
	"github.com/ecodeclub/webook/internal/payment/internal/service"
	"github.com/ecodeclub/webook/internal/payment/internal/service/wechat"
	"github.com/ecodeclub/webook/internal/payment/internal/web"
	"github.com/ecodeclub/webook/internal/payment/ioc"
	"github.com/ecodeclub/webook/internal/pkg/sequencenumber"
	"github.com/ego-component/egorm"
	"github.com/google/wire"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"gorm.io/gorm"
)

func InitModule(db *egorm.Component,
	mq mq.MQ,
	c ecache.Cache,
	um *user.Module,
	cm *credit.Module) (*Module, error) {
	wire.Build(

		ioc.InitWechatConfig,

		// 构建Svc
		// 构造NativePaymentService
		ioc.InitWechatClient,
		ioc.InitNativeApiService,
		wire.Bind(new(wechat.NativeAPIService), new(*native.NativeApiService)),
		ioc.InitWechatNativePaymentService,
		// 构造JSAPaymentService
		ioc.InitJSApiService,
		wire.Bind(new(wechat.JSAPIService), new(*jsapi.JsapiApiService)),
		ioc.InitWechatJSAPIPaymentService,
		newPaymentServices,

		wire.FieldsOf(new(*user.Module), "Svc"),
		wire.FieldsOf(new(*credit.Module), "Svc"),
		sequencenumber.NewGenerator,
		initDAO,
		repository.NewPaymentRepository,
		event.NewPaymentEventProducer,
		service.NewService,

		// 构建Hdl
		ioc.InitWechatNotifyHandler,
		wire.Bind(new(wechat.NotifyHandler), new(*notify.Handler)),

		web.NewHandler,

		// 构建SyncWechatOrderJob
		initSyncWechatOrderJob,

		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
}

func newPaymentServices(n *wechat.NativePaymentService, j *wechat.JSAPIPaymentService) map[ChannelType]service.PaymentService {
	return map[ChannelType]service.PaymentService{
		ChannelTypeWechat:   n,
		ChannelTypeWechatJS: j,
	}
}

var (
	once       = &sync.Once{}
	paymentDAO dao.PaymentDAO
)

func initDAO(db *gorm.DB) dao.PaymentDAO {
	once.Do(func() {
		_ = dao.InitTables(db)
		paymentDAO = dao.NewPaymentGORMDAO(db)
	})
	return paymentDAO
}

func initSyncWechatOrderJob(svc service.Service) *SyncWechatOrderJob {
	minutes := int64(30)
	seconds := int64(10)
	limit := 100
	return job.NewSyncWechatOrderJob(svc, minutes, seconds, limit)
}
