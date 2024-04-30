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

	"github.com/ecodeclub/ecache"
	"github.com/ecodeclub/mq-api"
	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/payment/internal/domain"
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
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"gorm.io/gorm"
)

type Handler = web.Handler
type Payment = domain.Payment
type Record = domain.PaymentRecord
type Channel = domain.PaymentChannel
type ChannelType = domain.ChannelType
type Service = service.Service
type SyncWechatOrderJob = job.SyncWechatOrderJob

const ChannelTypeCredit = domain.ChannelTypeCredit
const ChannelTypeWechat = domain.ChannelTypeWechat
const StatusPaidSuccess = domain.PaymentStatusPaidSuccess
const StatusFailed = domain.PaymentStatusPaidFailed

func InitModule(db *egorm.Component,
	mq mq.MQ,
	c ecache.Cache,
	cm *credit.Module) (*Module, error) {
	wire.Build(
		ioc.InitWechatNativeService,
		ioc.InitWechatConfig,
		ioc.InitWechatNotifyHandler,
		convertToNotifyHandler,
		ioc.InitWechatClient,
		ioc.InitNativeApiService,
		convertToNativeAPIService,
		initDAO,
		initPaymentEventProducer,
		web.NewHandler,
		service.NewService,
		repository.NewPaymentRepository,
		sequencenumber.NewGenerator,
		initSyncWechatOrderJob,
		wire.FieldsOf(new(*credit.Module), "Svc"),
		wire.Struct(new(Module), "*"),
	)
	return new(Module), nil
}

func convertToNotifyHandler(h *notify.Handler) wechat.NotifyHandler {
	return h
}

func convertToNativeAPIService(n *native.NativeApiService) wechat.NativeAPIService {
	return n
}

var (
	once       = &sync.Once{}
	paymentDAO dao.PaymentDAO
)

func initPaymentEventProducer(mq mq.MQ) (event.PaymentEventProducer, error) {
	p, err := mq.Producer("payment_events")
	if err != nil {
		return nil, err
	}
	return event.NewPaymentEventProducer(p)
}

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
