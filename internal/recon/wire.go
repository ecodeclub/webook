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

package recon

import (
	"time"

	"github.com/ecodeclub/webook/internal/credit"
	"github.com/ecodeclub/webook/internal/order"
	"github.com/ecodeclub/webook/internal/payment"
	"github.com/ecodeclub/webook/internal/recon/internal/job"
	"github.com/ecodeclub/webook/internal/recon/internal/service"
	"github.com/google/wire"
)

type (
	Service                = service.Service
	SyncPaymentAndOrderJob = job.SyncPaymentAndOrderJob
)

func InitModule(o *order.Module, p *payment.Module, c *credit.Module) (*Module, error) {
	wire.Build(
		initService,
		initSyncPaymentAndOrderJob,
		wire.FieldsOf(new(*order.Module), "Svc"),
		wire.FieldsOf(new(*payment.Module), "Svc"),
		wire.FieldsOf(new(*credit.Module), "Svc"),
		wire.Struct(new(Module), "*"),
	)
	return nil, nil
}

func initService(orderSvc order.Service,
	paymentSvc payment.Service,
	creditSvc credit.Service) Service {
	initialInterval := 100 * time.Millisecond
	maxInterval := 1 * time.Second
	maxRetries := int32(6)
	return service.NewService(orderSvc, paymentSvc, creditSvc, initialInterval, maxInterval, maxRetries)
}

func initSyncPaymentAndOrderJob(svc service.Service) *SyncPaymentAndOrderJob {
	minutes := int64(30)
	seconds := int64(10)
	limit := 100
	return job.NewSyncPaymentAndOrderJob(svc, minutes, seconds, limit)
}
